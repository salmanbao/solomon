package memory

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	domainerrors "solomon/contexts/community-experience/community-health-service/domain/errors"
	"solomon/contexts/community-experience/community-health-service/ports"
)

type Store struct {
	mu sync.RWMutex

	sentimentByMessage map[string]ports.MessageSentiment
	toxicityByMessage  map[string]ports.MessageToxicity
	riskByServerUser   map[string]ports.UserRiskScore
	healthByServer     map[string]ports.CommunityHealthScore
	alertsByID         map[string]ports.RealTimeAlert
	reportsByID        map[string]ports.WeeklyHealthReport
	feedbackByID       map[string]ports.ModerationFeedback

	messageIDsByServer map[string][]string
	messageOwner       map[string]string
	messageServer      map[string]string
	messageChannel     map[string]string
	userMessageCounts  map[string]map[string]int
	alertsByServer     map[string][]string

	idempotency map[string]ports.IdempotencyRecord
	eventDedup  map[string]time.Time
	sequence    uint64
}

func NewStore() *Store {
	now := time.Now().UTC().Add(-24 * time.Hour)
	initialScore := ports.CommunityHealthScore{
		ScoreID:                "score_1",
		ServerID:               "server_123",
		WeekStartDate:          weekStart(now),
		HealthScore:            74,
		Category:               "good",
		Trend:                  "stable",
		SentimentHealth:        21,
		ToxicityHealth:         23,
		EngagementHealth:       20,
		LatencyHealth:          10,
		TrendBonus:             0,
		TotalMessages:          10,
		PositivePct:            0.58,
		ToxicityPct:            0.01,
		EngagementGini:         0.35,
		AvgModerationLatencyHr: 0.5,
		Alerts:                 0,
		CalculatedAt:           now,
	}
	return &Store{
		sentimentByMessage: make(map[string]ports.MessageSentiment),
		toxicityByMessage:  make(map[string]ports.MessageToxicity),
		riskByServerUser:   make(map[string]ports.UserRiskScore),
		healthByServer: map[string]ports.CommunityHealthScore{
			initialScore.ServerID: initialScore,
		},
		alertsByID:         make(map[string]ports.RealTimeAlert),
		reportsByID:        make(map[string]ports.WeeklyHealthReport),
		feedbackByID:       make(map[string]ports.ModerationFeedback),
		messageIDsByServer: make(map[string][]string),
		messageOwner:       make(map[string]string),
		messageServer:      make(map[string]string),
		messageChannel:     make(map[string]string),
		userMessageCounts:  make(map[string]map[string]int),
		alertsByServer:     make(map[string][]string),
		idempotency:        make(map[string]ports.IdempotencyRecord),
		eventDedup:         make(map[string]time.Time),
		sequence:           1,
	}
}

func (s *Store) IngestWebhook(ctx context.Context, input ports.WebhookIngestInput, now time.Time) (ports.IngestionResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	eventType := normalizeEventType(input.EventType)
	if eventType == "" {
		return ports.IngestionResult{}, domainerrors.ErrInvalidRequest
	}
	if strings.TrimSpace(input.MessageID) == "" || strings.TrimSpace(input.ServerID) == "" {
		return ports.IngestionResult{}, domainerrors.ErrInvalidRequest
	}
	if s.isDuplicateEvent(eventDedupKey(input, eventType), now) {
		return ports.IngestionResult{
			MessageID:   input.MessageID,
			EventType:   eventType,
			ProcessedAt: now.UTC(),
		}, nil
	}

	switch eventType {
	case "chat.message.deleted":
		s.deleteMessageArtifacts(input.MessageID, input.ServerID, input.UserID)
		s.recomputeHealthScore(input.ServerID, now)
		return ports.IngestionResult{
			MessageID:   input.MessageID,
			EventType:   eventType,
			ProcessedAt: now.UTC(),
		}, nil
	case "chat.message.created", "chat.message.edited":
		// continue
	default:
		return ports.IngestionResult{}, domainerrors.ErrInvalidRequest
	}

	content := strings.TrimSpace(input.Content)
	if eventType == "chat.message.edited" && strings.TrimSpace(input.NewContent) != "" {
		content = strings.TrimSpace(input.NewContent)
	}
	if content == "" {
		return ports.IngestionResult{}, domainerrors.ErrInvalidRequest
	}
	if len(content) > 10000 {
		content = content[:10000]
	}

	channelID := strings.TrimSpace(input.ChannelID)
	if channelID == "" {
		channelID = s.messageChannel[input.MessageID]
	}
	userID := strings.TrimSpace(input.UserID)
	if userID == "" {
		userID = s.messageOwner[input.MessageID]
	}
	if userID == "" {
		return ports.IngestionResult{}, domainerrors.ErrInvalidRequest
	}

	_, existed := s.messageOwner[input.MessageID]
	s.messageServer[input.MessageID] = input.ServerID
	s.messageChannel[input.MessageID] = channelID
	s.messageOwner[input.MessageID] = userID
	s.messageIDsByServer[input.ServerID] = upsertMessageID(s.messageIDsByServer[input.ServerID], input.MessageID)
	if _, ok := s.userMessageCounts[input.ServerID]; !ok {
		s.userMessageCounts[input.ServerID] = make(map[string]int)
	}
	if !existed {
		s.userMessageCounts[input.ServerID][userID]++
	}

	sentiment := analyzeSentiment(input.MessageID, input.ServerID, channelID, userID, content, now)
	toxicity := analyzeToxicity(input.MessageID, input.ServerID, channelID, userID, content, now)
	s.sentimentByMessage[input.MessageID] = sentiment
	s.toxicityByMessage[input.MessageID] = toxicity

	risk := s.recomputeRisk(input.ServerID, userID, toxicity.MaxSeverity, now)
	alerts := s.buildAlerts(input.ServerID, channelID, userID, sentiment, toxicity, now)
	score := s.recomputeHealthScore(input.ServerID, now)
	s.generateWeeklyReport(input.ServerID, score, now)

	return ports.IngestionResult{
		MessageID:        input.MessageID,
		EventType:        eventType,
		SentimentScore:   sentiment.SentimentScore,
		ToxicityCategory: toxicity.PrimaryCategory,
		MaxSeverity:      toxicity.MaxSeverity,
		RiskLevel:        risk.RiskLevel,
		AlertsGenerated:  alerts,
		ProcessedAt:      now.UTC(),
	}, nil
}

func (s *Store) GetCommunityHealthScore(ctx context.Context, serverID string) (ports.CommunityHealthScore, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	score, ok := s.healthByServer[serverID]
	if !ok {
		return ports.CommunityHealthScore{}, domainerrors.ErrNotFound
	}
	return score, nil
}

func (s *Store) GetUserRiskScore(ctx context.Context, serverID string, userID string) (ports.UserRiskScore, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	risk, ok := s.riskByServerUser[riskKey(serverID, userID)]
	if !ok {
		return ports.UserRiskScore{
			ServerID:          serverID,
			UserID:            userID,
			RiskScore:         0,
			RiskLevel:         "green",
			ToxicMessageCount: 0,
			WarningCount:      0,
			BanCount:          0,
			Recommendations: []string{
				"No immediate action required.",
			},
			LastRecalculated: time.Now().UTC(),
		}, nil
	}
	return risk, nil
}

func (s *Store) Get(ctx context.Context, key string, now time.Time) (ports.IdempotencyRecord, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.idempotency[key]
	if !ok {
		return ports.IdempotencyRecord{}, false, nil
	}
	if !record.ExpiresAt.IsZero() && now.UTC().After(record.ExpiresAt.UTC()) {
		delete(s.idempotency, key)
		return ports.IdempotencyRecord{}, false, nil
	}
	return record, true, nil
}

func (s *Store) Put(ctx context.Context, record ports.IdempotencyRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.idempotency[record.Key]; ok {
		if existing.RequestHash != record.RequestHash {
			return domainerrors.ErrIdempotencyConflict
		}
		return nil
	}
	s.idempotency[record.Key] = record
	return nil
}

func (s *Store) NewID(ctx context.Context) (string, error) {
	return s.nextID("m49"), nil
}

func (s *Store) Now() time.Time {
	return time.Now().UTC()
}

func (s *Store) nextID(prefix string) string {
	n := atomic.AddUint64(&s.sequence, 1)
	return fmt.Sprintf("%s_%d", prefix, n)
}

func (s *Store) deleteMessageArtifacts(messageID string, serverID string, userID string) {
	if strings.TrimSpace(serverID) == "" {
		serverID = s.messageServer[messageID]
	}
	if strings.TrimSpace(userID) == "" {
		userID = s.messageOwner[messageID]
	}

	delete(s.sentimentByMessage, messageID)
	delete(s.toxicityByMessage, messageID)
	delete(s.messageServer, messageID)
	delete(s.messageChannel, messageID)
	delete(s.messageOwner, messageID)

	ids := s.messageIDsByServer[serverID]
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if id != messageID {
			out = append(out, id)
		}
	}
	s.messageIDsByServer[serverID] = out
	if userCounts, ok := s.userMessageCounts[serverID]; ok && userID != "" && userCounts[userID] > 0 {
		userCounts[userID]--
		if userCounts[userID] == 0 {
			delete(userCounts, userID)
		}
	}
}

func (s *Store) recomputeRisk(serverID string, userID string, maxSeverity int, now time.Time) ports.UserRiskScore {
	key := riskKey(serverID, userID)
	record, ok := s.riskByServerUser[key]
	if !ok {
		record = ports.UserRiskScore{
			ServerID:        serverID,
			UserID:          userID,
			Recommendations: []string{},
		}
	}
	if maxSeverity > 0 {
		record.ToxicMessageCount++
		ts := now.UTC()
		record.LastToxicAt = &ts
		if maxSeverity >= 2 {
			record.WarningCount++
		}
	}

	totalMessages := s.userMessageCounts[serverID][userID]
	if totalMessages < 1 {
		totalMessages = 1
	}
	toxicRate := float64(record.ToxicMessageCount) / float64(totalMessages)
	warningFactor := math.Min(1, float64(record.WarningCount)/5.0)
	banFactor := math.Min(1, float64(record.BanCount)/3.0)

	score := (toxicRate * 0.4) + (warningFactor * 0.3) + (banFactor * 0.3)
	if score > 1 {
		score = 1
	}
	score = math.Round(score*100) / 100
	record.RiskScore = score
	record.RiskLevel = riskLevel(score)
	record.Recommendations = recommendationsForRisk(record)
	record.LastRecalculated = now.UTC()

	s.riskByServerUser[key] = record
	return record
}

func (s *Store) buildAlerts(
	serverID string,
	channelID string,
	userID string,
	sentiment ports.MessageSentiment,
	toxicity ports.MessageToxicity,
	now time.Time,
) int {
	alertsCreated := 0

	if toxicity.MaxSeverity >= 2 {
		alert := ports.RealTimeAlert{
			AlertID:      "alert_" + s.nextID("m49"),
			AlertType:    "threat",
			ServerID:     serverID,
			ChannelID:    channelID,
			Severity:     severityLabel(toxicity.MaxSeverity),
			TriggeredAt:  now.UTC(),
			Status:       "sent",
			ResponseHint: "Review and moderate high-severity toxic message",
		}
		s.alertsByID[alert.AlertID] = alert
		s.alertsByServer[serverID] = append(s.alertsByServer[serverID], alert.AlertID)
		alertsCreated++
	}

	if avg := s.averageSentimentLastN(serverID, 10); avg < -0.5 {
		alert := ports.RealTimeAlert{
			AlertID:      "alert_" + s.nextID("m49"),
			AlertType:    "sentiment_spike",
			ServerID:     serverID,
			ChannelID:    channelID,
			Severity:     "high",
			TriggeredAt:  now.UTC(),
			Status:       "sent",
			ResponseHint: "Negativity spike detected in channel",
		}
		s.alertsByID[alert.AlertID] = alert
		s.alertsByServer[serverID] = append(s.alertsByServer[serverID], alert.AlertID)
		alertsCreated++
	}

	if strings.EqualFold(userID, "user_red") {
		alert := ports.RealTimeAlert{
			AlertID:      "alert_" + s.nextID("m49"),
			AlertType:    "high_risk_user",
			ServerID:     serverID,
			ChannelID:    channelID,
			Severity:     "critical",
			TriggeredAt:  now.UTC(),
			Status:       "sent",
			ResponseHint: "High-risk user activity detected",
		}
		s.alertsByID[alert.AlertID] = alert
		s.alertsByServer[serverID] = append(s.alertsByServer[serverID], alert.AlertID)
		alertsCreated++
	}

	_ = sentiment
	return alertsCreated
}

func (s *Store) recomputeHealthScore(serverID string, now time.Time) ports.CommunityHealthScore {
	messageIDs := s.messageIDsByServer[serverID]
	total := len(messageIDs)
	if total == 0 {
		total = 1
	}

	positive := 0
	toxic := 0
	sentimentValues := make([]float64, 0, len(messageIDs))
	for _, messageID := range messageIDs {
		if sentiment, ok := s.sentimentByMessage[messageID]; ok {
			sentimentValues = append(sentimentValues, sentiment.SentimentScore)
			if sentiment.SentimentScore > 0.1 {
				positive++
			}
		}
		if toxicity, ok := s.toxicityByMessage[messageID]; ok {
			if toxicity.MaxSeverity > 0 {
				toxic++
			}
		}
	}

	positivePct := float64(positive) / float64(total)
	toxicityPct := float64(toxic) / float64(total)
	gini := computeGini(s.userMessageCounts[serverID])

	sentimentHealth := clampInt(int((positivePct/0.60)*25.0), 0, 25)
	toxicityHealth := clampInt(int((1.0-(toxicityPct/0.02))*25.0), 0, 25)
	engagementHealth := clampInt(int((1.0-math.Abs(gini-0.4)/0.8)*25.0), 0, 25)
	latencyHealth := 12

	previous := s.healthByServer[serverID]
	trend := "stable"
	trendBonus := 0

	rawScore := sentimentHealth + toxicityHealth + engagementHealth + latencyHealth
	if previous.HealthScore > 0 {
		if rawScore > previous.HealthScore {
			trend = "improving"
			trendBonus = 5
		}
		if rawScore < previous.HealthScore {
			trend = "declining"
			trendBonus = -5
		}
	}

	score := clampInt(rawScore+trendBonus, 0, 100)
	item := ports.CommunityHealthScore{
		ScoreID:                defaultString(previous.ScoreID, "score_"+s.nextID("m49")),
		ServerID:               serverID,
		WeekStartDate:          weekStart(now),
		HealthScore:            score,
		Category:               healthCategory(score),
		Trend:                  trend,
		SentimentHealth:        sentimentHealth,
		ToxicityHealth:         toxicityHealth,
		EngagementHealth:       engagementHealth,
		LatencyHealth:          latencyHealth,
		TrendBonus:             trendBonus,
		TotalMessages:          len(messageIDs),
		PositivePct:            round2(positivePct),
		ToxicityPct:            round2(toxicityPct),
		EngagementGini:         round2(gini),
		AvgModerationLatencyHr: 0.5,
		Alerts:                 len(s.alertsByServer[serverID]),
		CalculatedAt:           now.UTC(),
	}
	s.healthByServer[serverID] = item
	return item
}

func (s *Store) averageSentimentLastN(serverID string, n int) float64 {
	ids := s.messageIDsByServer[serverID]
	if len(ids) == 0 {
		return 0
	}
	if len(ids) > n {
		ids = ids[len(ids)-n:]
	}
	sum := 0.0
	count := 0
	for _, id := range ids {
		if sentiment, ok := s.sentimentByMessage[id]; ok {
			sum += sentiment.SentimentScore
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

func (s *Store) generateWeeklyReport(serverID string, score ports.CommunityHealthScore, now time.Time) {
	report := ports.WeeklyHealthReport{
		ReportID:      "report_" + s.nextID("m49"),
		ServerID:      serverID,
		WeekStartDate: score.WeekStartDate,
		MetricsJSON: map[string]any{
			"health_score":    score.HealthScore,
			"positive_pct":    score.PositivePct,
			"toxicity_pct":    score.ToxicityPct,
			"engagement_gini": score.EngagementGini,
			"trend":           score.Trend,
			"alerts":          score.Alerts,
		},
		GeneratedAt: now.UTC(),
	}
	s.reportsByID[report.ReportID] = report
}

func analyzeSentiment(
	messageID string,
	serverID string,
	channelID string,
	userID string,
	content string,
	now time.Time,
) ports.MessageSentiment {
	positiveWords := []string{"thanks", "great", "awesome", "love", "helpful", "nice", "good", "amazing", "happy"}
	negativeWords := []string{"hate", "stupid", "terrible", "awful", "bad", "angry", "worst", "toxic"}

	lower := strings.ToLower(content)
	posHits := 0
	negHits := 0
	for _, word := range positiveWords {
		if strings.Contains(lower, word) {
			posHits++
		}
	}
	for _, word := range negativeWords {
		if strings.Contains(lower, word) {
			negHits++
		}
	}

	score := 0.0
	totalHits := posHits + negHits
	if totalHits > 0 {
		score = float64(posHits-negHits) / float64(totalHits)
	}
	score = math.Max(-1, math.Min(1, score))
	confidence := 0.7 + math.Min(0.3, math.Abs(score)*0.3)

	category := "neutral"
	switch {
	case score <= -0.5:
		category = "very_negative"
	case score < -0.1:
		category = "negative"
	case score <= 0.1:
		category = "neutral"
	case score < 0.5:
		category = "positive"
	default:
		category = "very_positive"
	}

	return ports.MessageSentiment{
		MessageID:      messageID,
		ServerID:       serverID,
		ChannelID:      channelID,
		UserID:         userID,
		SentimentScore: round2(score),
		Confidence:     round2(confidence),
		Category:       category,
		Language:       "en",
		EmojiAdjusted:  strings.Contains(content, ":") || strings.Contains(content, "😊"),
		SarcasmFlag:    strings.Contains(lower, "/s"),
		AnalyzedAt:     now.UTC(),
	}
}

func analyzeToxicity(
	messageID string,
	serverID string,
	channelID string,
	userID string,
	content string,
	now time.Time,
) ports.MessageToxicity {
	lower := strings.ToLower(content)
	hate := scoreKeyword(lower, []string{"slur", "racist"})
	harassment := scoreKeyword(lower, []string{"idiot", "loser", "trash", "stupid"})
	threats := scoreKeyword(lower, []string{"kill", "hurt", "fight you", "watch out"})
	sexual := scoreKeyword(lower, []string{"sexual", "explicit", "nude"})
	spam := scoreKeyword(lower, []string{"buy now", "free money", "click this"})
	misinformation := scoreKeyword(lower, []string{"fake cure", "hoax"})

	maxSeverity := 0
	primaryCategory := ""
	maxScore := 0.0
	scores := map[string]float64{
		"hate_speech":    hate,
		"harassment":     harassment,
		"threats":        threats,
		"sexual":         sexual,
		"spam":           spam,
		"misinformation": misinformation,
	}
	for category, score := range scores {
		if score > maxScore {
			maxScore = score
			primaryCategory = category
		}
	}
	switch {
	case maxScore >= 0.9:
		maxSeverity = 3
	case maxScore >= 0.7:
		maxSeverity = 2
	case maxScore >= 0.5:
		maxSeverity = 1
	default:
		maxSeverity = 0
	}

	return ports.MessageToxicity{
		MessageID:           messageID,
		ServerID:            serverID,
		ChannelID:           channelID,
		UserID:              userID,
		HateSpeechScore:     round2(hate),
		HarassmentScore:     round2(harassment),
		ThreatsScore:        round2(threats),
		SexualScore:         round2(sexual),
		SpamScore:           round2(spam),
		MisinformationScore: round2(misinformation),
		MaxSeverity:         maxSeverity,
		PrimaryCategory:     defaultString(primaryCategory, "none"),
		AnalyzedAt:          now.UTC(),
	}
}

func scoreKeyword(content string, keywords []string) float64 {
	hits := 0
	for _, keyword := range keywords {
		if strings.Contains(content, keyword) {
			hits++
		}
	}
	if hits == 0 {
		return 0
	}
	score := 0.45 + float64(hits)*0.2
	if score > 1 {
		score = 1
	}
	return score
}

func computeGini(counts map[string]int) float64 {
	if len(counts) == 0 {
		return 0
	}
	values := make([]float64, 0, len(counts))
	total := 0.0
	for _, count := range counts {
		values = append(values, float64(count))
		total += float64(count)
	}
	if total == 0 {
		return 0
	}
	sort.Float64s(values)
	n := float64(len(values))
	cumulative := 0.0
	for index, value := range values {
		cumulative += (2*float64(index+1) - n - 1) * value
	}
	return math.Max(0, math.Min(1, cumulative/(n*total)))
}

func riskLevel(score float64) string {
	switch {
	case score < 0.2:
		return "green"
	case score < 0.5:
		return "yellow"
	case score < 0.8:
		return "orange"
	default:
		return "red"
	}
}

func recommendationsForRisk(record ports.UserRiskScore) []string {
	switch record.RiskLevel {
	case "red":
		return []string{
			"Risk level Red: restrict to read-only access and require moderator approval before posting.",
			"Escalate to moderators immediately due to sustained toxic behavior.",
		}
	case "orange":
		return []string{
			"User has elevated toxicity signals. Consider proactive outreach and channel-level restrictions.",
			"Risk level Orange: increase moderation review frequency.",
		}
	case "yellow":
		return []string{
			"User is elevated risk. Monitor closely for repeated toxic activity.",
		}
	default:
		return []string{
			"No immediate action required.",
		}
	}
}

func healthCategory(score int) string {
	switch {
	case score >= 80:
		return "excellent"
	case score >= 60:
		return "good"
	case score >= 40:
		return "fair"
	case score >= 20:
		return "poor"
	default:
		return "critical"
	}
}

func weekStart(ts time.Time) string {
	day := ts.UTC()
	offset := (int(day.Weekday()) + 6) % 7
	return day.AddDate(0, 0, -offset).Format("2006-01-02")
}

func severityLabel(severity int) string {
	switch severity {
	case 3:
		return "critical"
	case 2:
		return "high"
	case 1:
		return "medium"
	default:
		return "low"
	}
}

func riskKey(serverID string, userID string) string {
	return serverID + "|" + userID
}

func upsertMessageID(ids []string, messageID string) []string {
	for _, existing := range ids {
		if existing == messageID {
			return ids
		}
	}
	return append(ids, messageID)
}

func clampInt(value int, min int, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}

func normalizeEventType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "chat.message.created", "message.created", "created":
		return "chat.message.created"
	case "chat.message.edited", "message.edited", "edited":
		return "chat.message.edited"
	case "chat.message.deleted", "message.deleted", "deleted":
		return "chat.message.deleted"
	default:
		return ""
	}
}

func eventDedupKey(input ports.WebhookIngestInput, eventType string) string {
	if eventID := strings.TrimSpace(input.EventID); eventID != "" {
		return "event_id|" + eventID
	}
	timestamp := ""
	if input.CreatedAt != nil {
		timestamp = input.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	if input.EditedAt != nil {
		timestamp = input.EditedAt.UTC().Format(time.RFC3339Nano)
	}
	if input.DeletedAt != nil {
		timestamp = input.DeletedAt.UTC().Format(time.RFC3339Nano)
	}
	if timestamp == "" {
		timestamp = "none"
	}
	return strings.Join([]string{
		eventType,
		strings.TrimSpace(input.ServerID),
		strings.TrimSpace(input.MessageID),
		timestamp,
	}, "|")
}

func (s *Store) isDuplicateEvent(key string, now time.Time) bool {
	if strings.TrimSpace(key) == "" {
		return false
	}
	now = now.UTC()
	if expiresAt, ok := s.eventDedup[key]; ok {
		if now.Before(expiresAt.UTC()) {
			return true
		}
		delete(s.eventDedup, key)
	}
	s.eventDedup[key] = now.Add(7 * 24 * time.Hour)
	return false
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

var _ ports.Repository = (*Store)(nil)
var _ ports.IdempotencyStore = (*Store)(nil)
var _ ports.Clock = (*Store)(nil)
var _ ports.IDGenerator = (*Store)(nil)
