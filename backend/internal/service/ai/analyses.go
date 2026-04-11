package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AnalysisService orchestrates AI analyses using DeepSeek.
type AnalysisService struct {
	client *DeepSeekClient
	db     *pgxpool.Pool
}

func NewAnalysisService(client *DeepSeekClient, db *pgxpool.Pool) *AnalysisService {
	return &AnalysisService{client: client, db: db}
}

// ── Result Types ─────────────────────────────────

type CollaborationRec struct {
	ResearcherID   string   `json:"researcher_id"`
	ResearcherName string   `json:"researcher_name"`
	MatchScore     float64  `json:"match_score"`
	ReasonAr       string   `json:"reason_ar"`
	ReasonEn       string   `json:"reason_en"`
	PotentialTopics []string `json:"potential_topics"`
}

type TrendingTopic struct {
	TopicAr    string  `json:"topic_ar"`
	TopicEn    string  `json:"topic_en"`
	Direction  string  `json:"direction"` // rising, declining, stable
	Score      float64 `json:"score"`
	Researchers int    `json:"researchers"`
	Publications int   `json:"publications"`
}

type StrategyAdvice struct {
	RecommendedJournals []struct {
		Name   string `json:"name"`
		Reason string `json:"reason"`
	} `json:"recommended_journals"`
	TopicsToFocus   []string `json:"topics_to_focus"`
	CoauthorStrategy string  `json:"coauthor_strategy"`
	ActionItems      []string `json:"action_items"`
	SummaryAr        string   `json:"summary_ar"`
	SummaryEn        string   `json:"summary_en"`
}

type GapAnalysisResult struct {
	UnderrepresentedFields []struct {
		FieldAr     string  `json:"field_ar"`
		FieldEn     string  `json:"field_en"`
		GlobalGrowth float64 `json:"global_growth_pct"`
		LocalOutput  int     `json:"local_output"`
		Opportunity  string  `json:"opportunity"`
	} `json:"underrepresented_fields"`
	RecommendedPrograms []struct {
		ProgramAr string `json:"program_ar"`
		ProgramEn string `json:"program_en"`
		Rationale string `json:"rationale"`
	} `json:"recommended_programs"`
	SummaryAr string `json:"summary_ar"`
	SummaryEn string `json:"summary_en"`
}

// ── Collaboration Recommendations ────────────────

func (s *AnalysisService) CollaborationRecommendations(ctx context.Context, researcherID, requestedBy string) ([]CollaborationRec, error) {
	// Build context: target researcher + candidates
	var targetName, targetFields string
	s.db.QueryRow(ctx, `
		SELECT name_ar, COALESCE(array_to_string(ARRAY(
			SELECT title_en FROM publications p
			JOIN publication_authors pa ON pa.publication_id = p.id
			WHERE pa.researcher_id = $1 LIMIT 5
		), '; '), '')
		FROM researchers WHERE id = $1
	`, researcherID).Scan(&targetName, &targetFields)

	rows, _ := s.db.Query(ctx, `
		SELECT id, name_ar, COALESCE(name_en, ''), COALESCE(title_ar, ''), h_index, total_publications
		FROM researchers WHERE id != $1 AND status = 'active'
		ORDER BY h_index DESC LIMIT 30
	`, researcherID)
	defer rows.Close()

	var candidates []string
	for rows.Next() {
		var id, nameAr, nameEn, title string
		var hIndex, pubs int
		rows.Scan(&id, &nameAr, &nameEn, &title, &hIndex, &pubs)
		candidates = append(candidates, fmt.Sprintf("ID:%s | %s (%s) | %s | h:%d | pubs:%d", id, nameAr, nameEn, title, hIndex, pubs))
	}

	systemPrompt := `أنت محلل بحثي متخصص. مهمتك اقتراح باحثين مناسبين للتعاون البحثي.
أجب بصيغة JSON array فقط بدون أي نص إضافي. كل عنصر يحتوي:
researcher_id, researcher_name, match_score (0-1), reason_ar, reason_en, potential_topics (array)`

	userPrompt := fmt.Sprintf(`الباحث المستهدف: %s
آخر أبحاثه: %s

الباحثون المتاحون للتعاون:
%s

اقترح أفضل 5 باحثين للتعاون مع شرح السبب ومواضيع مقترحة.`, targetName, targetFields, strings.Join(candidates, "\n"))

	var results []CollaborationRec
	tokens, err := s.client.ChatJSON(ctx, "deepseek-chat", systemPrompt, userPrompt, &results)
	if err != nil {
		return nil, err
	}

	s.saveAnalysis(ctx, "collaboration_rec", "researcher", researcherID, requestedBy, results, tokens)
	return results, nil
}

// ── Trending Topics ──────────────────────────────

func (s *AnalysisService) TrendingTopics(ctx context.Context, requestedBy string) ([]TrendingTopic, error) {
	rows, _ := s.db.Query(ctx, `
		SELECT COALESCE(p.title_en, p.title_ar, '') AS title,
		       p.citation_count,
		       EXTRACT(YEAR FROM p.publication_date)::int AS year
		FROM publications p
		WHERE p.publication_date >= CURRENT_DATE - INTERVAL '2 years'
		ORDER BY p.citation_count DESC
		LIMIT 50
	`)
	defer rows.Close()

	var pubSummaries []string
	for rows.Next() {
		var title string
		var citations, year int
		rows.Scan(&title, &citations, &year)
		pubSummaries = append(pubSummaries, fmt.Sprintf("[%d] %s (cit:%d)", year, title, citations))
	}

	systemPrompt := `أنت محلل اتجاهات بحثية. حلل الأبحاث التالية وحدد المواضيع الرائجة والمتراجعة.
أجب بصيغة JSON array فقط. كل عنصر: topic_ar, topic_en, direction (rising/declining/stable), score (0-1), researchers (عدد تقديري), publications (عدد)`

	userPrompt := fmt.Sprintf("أبحاث المؤسسة في آخر سنتين:\n%s\n\nحدد أهم 8 مواضيع رائجة ومتراجعة.", strings.Join(pubSummaries, "\n"))

	var results []TrendingTopic
	tokens, err := s.client.ChatJSON(ctx, "deepseek-chat", systemPrompt, userPrompt, &results)
	if err != nil {
		return nil, err
	}

	s.saveAnalysis(ctx, "trend_analysis", "institution", "all", requestedBy, results, tokens)
	return results, nil
}

// ── Publication Strategy ─────────────────────────

func (s *AnalysisService) PublicationStrategy(ctx context.Context, researcherID, requestedBy string) (*StrategyAdvice, error) {
	var nameAr string
	var hIndex, totalPubs, totalCit int
	s.db.QueryRow(ctx, `SELECT name_ar, h_index, total_publications, total_citations FROM researchers WHERE id = $1`,
		researcherID).Scan(&nameAr, &hIndex, &totalPubs, &totalCit)

	rows, _ := s.db.Query(ctx, `
		SELECT COALESCE(p.title_en, p.title_ar, ''), p.citation_count,
		       COALESCE(j.name, ''), COALESCE(j.sjr_ranking::text, 'unknown')
		FROM publications p
		JOIN publication_authors pa ON pa.publication_id = p.id
		LEFT JOIN journals j ON j.id = p.journal_id
		WHERE pa.researcher_id = $1
		ORDER BY p.publication_date DESC LIMIT 15
	`, researcherID)
	defer rows.Close()

	var pubHistory []string
	for rows.Next() {
		var title, journal, ranking string
		var cit int
		rows.Scan(&title, &cit, &journal, &ranking)
		pubHistory = append(pubHistory, fmt.Sprintf("%s | journal:%s (%s) | cit:%d", title, journal, ranking, cit))
	}

	systemPrompt := `أنت مستشار نشر أكاديمي. قدم نصائح مخصصة لتحسين الأداء البحثي.
أجب بصيغة JSON object فقط:
{
  "recommended_journals": [{"name": "...", "reason": "..."}],
  "topics_to_focus": ["..."],
  "coauthor_strategy": "...",
  "action_items": ["..."],
  "summary_ar": "...",
  "summary_en": "..."
}`

	userPrompt := fmt.Sprintf(`الباحث: %s | h-index: %d | أبحاث: %d | استشهادات: %d

تاريخ النشر:
%s

قدم نصائح مخصصة: مجلات مستهدفة، مواضيع بحثية، استراتيجية تعاون.`, nameAr, hIndex, totalPubs, totalCit, strings.Join(pubHistory, "\n"))

	var result StrategyAdvice
	tokens, err := s.client.ChatJSON(ctx, "deepseek-chat", systemPrompt, userPrompt, &result)
	if err != nil {
		return nil, err
	}

	s.saveAnalysis(ctx, "publication_strategy", "researcher", researcherID, requestedBy, result, tokens)
	return &result, nil
}

// ── Gap Analysis ─────────────────────────────────

func (s *AnalysisService) GapAnalysis(ctx context.Context, requestedBy string) (*GapAnalysisResult, error) {
	rows, _ := s.db.Query(ctx, `
		SELECT rf.name_en, rf.name_ar, COUNT(pf.publication_id)
		FROM research_fields rf
		LEFT JOIN publication_fields pf ON pf.field_id = rf.id
		GROUP BY rf.id, rf.name_en, rf.name_ar
		ORDER BY COUNT(pf.publication_id) DESC
		LIMIT 30
	`)
	defer rows.Close()

	var fieldSummaries []string
	for rows.Next() {
		var nameEn, nameAr string
		var count int
		rows.Scan(&nameEn, &nameAr, &count)
		fieldSummaries = append(fieldSummaries, fmt.Sprintf("%s (%s): %d أبحاث", nameAr, nameEn, count))
	}

	// Also get overall stats
	var totalPubs, totalResearchers int
	s.db.QueryRow(ctx, `SELECT COUNT(*) FROM publications`).Scan(&totalPubs)
	s.db.QueryRow(ctx, `SELECT COUNT(*) FROM researchers WHERE status = 'active'`).Scan(&totalResearchers)

	systemPrompt := `أنت محلل استراتيجي للبحث العلمي. حلل الفجوات البحثية واقترح مجالات جديدة.
أجب بصيغة JSON object فقط:
{
  "underrepresented_fields": [{"field_ar": "...", "field_en": "...", "global_growth_pct": 0, "local_output": 0, "opportunity": "..."}],
  "recommended_programs": [{"program_ar": "...", "program_en": "...", "rationale": "..."}],
  "summary_ar": "...",
  "summary_en": "..."
}`

	userPrompt := fmt.Sprintf(`المؤسسة لديها %d باحث و %d بحث منشور.

المجالات البحثية الحالية:
%s

حلل الفجوات واقترح مجالات بحثية جديدة ذات نمو عالمي عالٍ يمكن للمؤسسة التوسع فيها.`,
		totalResearchers, totalPubs, strings.Join(fieldSummaries, "\n"))

	var result GapAnalysisResult
	tokens, err := s.client.ChatJSON(ctx, "deepseek-reasoner", systemPrompt, userPrompt, &result)
	if err != nil {
		// Fallback to deepseek-chat if reasoner fails
		tokens, err = s.client.ChatJSON(ctx, "deepseek-chat", systemPrompt, userPrompt, &result)
		if err != nil {
			return nil, err
		}
	}

	s.saveAnalysis(ctx, "gap_analysis", "institution", "all", requestedBy, result, tokens)
	return &result, nil
}

// ── Save analysis to DB ──────────────────────────

func (s *AnalysisService) saveAnalysis(ctx context.Context, analysisType, targetType, targetID, requestedBy string, result any, tokens int) {
	resultJSON, _ := json.Marshal(result)

	_, err := s.db.Exec(ctx, `
		INSERT INTO ai_analyses (id, analysis_type, target_type, target_id, result, model_used, tokens_used, status, requested_by, expires_at)
		VALUES ($1, $2::analysis_type, $3::analysis_target, $4, $5, 'deepseek', $6, 'completed', $7, $8)
	`, uuid.New().String(), analysisType, targetType, targetID, resultJSON, tokens, nilStr(requestedBy),
		time.Now().Add(7*24*time.Hour))

	if err != nil {
		slog.Warn("failed to save AI analysis", "type", analysisType, "error", err)
	}
}

func nilStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// GetCachedAnalysis returns a cached analysis if still valid.
func (s *AnalysisService) GetCachedAnalysis(ctx context.Context, analysisType, targetType, targetID string) (json.RawMessage, bool) {
	var result json.RawMessage
	err := s.db.QueryRow(ctx, `
		SELECT result FROM ai_analyses
		WHERE analysis_type = $1::analysis_type AND target_type = $2::analysis_target AND target_id = $3
		  AND status = 'completed' AND expires_at > NOW()
		ORDER BY created_at DESC LIMIT 1
	`, analysisType, targetType, targetID).Scan(&result)

	if err != nil {
		return nil, false
	}
	return result, true
}
