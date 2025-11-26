package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/otelguard/otelguard/internal/repository/clickhouse"
	"go.uber.org/zap"
)

// ScoreAnalyticsService provides advanced analytics for scores
type ScoreAnalyticsService struct {
	evaluationRepo *clickhouse.EvaluationResultRepository
	logger         *zap.Logger
}

// NewScoreAnalyticsService creates a new score analytics service
func NewScoreAnalyticsService(
	evaluationRepo *clickhouse.EvaluationResultRepository,
	logger *zap.Logger,
) *ScoreAnalyticsService {
	return &ScoreAnalyticsService{
		evaluationRepo: evaluationRepo,
		logger:         logger,
	}
}

// ScoreDistribution represents score distribution statistics
type ScoreDistribution struct {
	ScoreName  string             `json:"scoreName"`
	Mean       float64            `json:"mean"`
	Median     float64            `json:"median"`
	StdDev     float64            `json:"stdDev"`
	Min        float64            `json:"min"`
	Max        float64            `json:"max"`
	Count      int                `json:"count"`
	Histogram  []HistogramBucket  `json:"histogram"`
	Percentiles map[string]float64 `json:"percentiles"`
}

// HistogramBucket represents a bucket in the histogram
type HistogramBucket struct {
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
	Count int     `json:"count"`
}

// CorrelationResult represents correlation between two scores
type CorrelationResult struct {
	Score1         string  `json:"score1"`
	Score2         string  `json:"score2"`
	Pearson        float64 `json:"pearson"`
	Spearman       float64 `json:"spearman"`
	SampleSize     int     `json:"sampleSize"`
	PValue         float64 `json:"pValue"`
	IsSignificant  bool    `json:"isSignificant"`
}

// ScoreBreakdown represents score statistics by dimension
type ScoreBreakdown struct {
	Dimension string                       `json:"dimension"`
	Values    map[string]BreakdownStats    `json:"values"`
}

// BreakdownStats contains statistics for a dimension value
type BreakdownStats struct {
	Count  int     `json:"count"`
	Mean   float64 `json:"mean"`
	StdDev float64 `json:"stdDev"`
}

// CohenKappaResult represents inter-annotator agreement
type CohenKappaResult struct {
	Kappa         float64            `json:"kappa"`
	Interpretation string            `json:"interpretation"`
	Agreement     float64            `json:"agreement"`
	ChanceAgreement float64          `json:"chanceAgreement"`
	ConfusionMatrix map[string]map[string]int `json:"confusionMatrix"`
}

// F1ScoreResult represents classification metrics
type F1ScoreResult struct {
	F1Score   float64 `json:"f1Score"`
	Precision float64 `json:"precision"`
	Recall    float64 `json:"recall"`
	Accuracy  float64 `json:"accuracy"`
	TP        int     `json:"truePositives"`
	TN        int     `json:"trueNegatives"`
	FP        int     `json:"falsePositives"`
	FN        int     `json:"falseNegatives"`
}

// ScoreTrend represents score trends over time
type ScoreTrend struct {
	ScoreName  string            `json:"scoreName"`
	Datapoints []TrendDatapoint  `json:"datapoints"`
	Trend      string            `json:"trend"` // increasing, decreasing, stable
	ChangeRate float64           `json:"changeRate"`
}

// TrendDatapoint represents a single point in the trend
type TrendDatapoint struct {
	Timestamp time.Time `json:"timestamp"`
	Mean      float64   `json:"mean"`
	Count     int       `json:"count"`
}

// GetScoreDistribution calculates distribution statistics for a score
func (s *ScoreAnalyticsService) GetScoreDistribution(
	ctx context.Context,
	projectID uuid.UUID,
	scoreName string,
	startTime, endTime time.Time,
) (*ScoreDistribution, error) {
	// This is a simplified implementation
	// In production, you would query the evaluation_results table

	// Mock data for demonstration
	values := []float64{0.7, 0.8, 0.75, 0.9, 0.65, 0.85, 0.78, 0.82, 0.88, 0.76}

	if len(values) == 0 {
		return nil, fmt.Errorf("no data available for score %s", scoreName)
	}

	// Sort values for percentile calculations
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	// Calculate statistics
	mean := calculateMean(values)
	median := calculatePercentile(sorted, 50)
	stdDev := calculateStdDev(values, mean)
	min := sorted[0]
	max := sorted[len(sorted)-1]

	// Create histogram (10 buckets)
	histogram := createHistogram(values, min, max, 10)

	// Calculate percentiles
	percentiles := map[string]float64{
		"p25": calculatePercentile(sorted, 25),
		"p50": median,
		"p75": calculatePercentile(sorted, 75),
		"p90": calculatePercentile(sorted, 90),
		"p95": calculatePercentile(sorted, 95),
		"p99": calculatePercentile(sorted, 99),
	}

	return &ScoreDistribution{
		ScoreName:   scoreName,
		Mean:        mean,
		Median:      median,
		StdDev:      stdDev,
		Min:         min,
		Max:         max,
		Count:       len(values),
		Histogram:   histogram,
		Percentiles: percentiles,
	}, nil
}

// GetCorrelation calculates correlation between two scores
func (s *ScoreAnalyticsService) GetCorrelation(
	ctx context.Context,
	projectID uuid.UUID,
	score1, score2 string,
	startTime, endTime time.Time,
) (*CorrelationResult, error) {
	// This is a simplified implementation
	// In production, you would query paired scores from the database

	// Mock paired data
	pairs1 := []float64{0.7, 0.8, 0.75, 0.9, 0.65}
	pairs2 := []float64{0.75, 0.82, 0.78, 0.88, 0.68}

	if len(pairs1) != len(pairs2) || len(pairs1) == 0 {
		return nil, fmt.Errorf("insufficient paired data")
	}

	pearson := calculatePearsonCorrelation(pairs1, pairs2)
	spearman := calculateSpearmanCorrelation(pairs1, pairs2)

	// Simple significance test (|r| > 0.5 with n > 10)
	isSignificant := math.Abs(pearson) > 0.5 && len(pairs1) > 10

	return &CorrelationResult{
		Score1:        score1,
		Score2:        score2,
		Pearson:       pearson,
		Spearman:      spearman,
		SampleSize:    len(pairs1),
		PValue:        0.05, // Simplified
		IsSignificant: isSignificant,
	}, nil
}

// GetScoreBreakdown gets score statistics broken down by dimension
func (s *ScoreAnalyticsService) GetScoreBreakdown(
	ctx context.Context,
	projectID uuid.UUID,
	scoreName string,
	dimension string, // e.g., "model", "user_id", "prompt_version"
	startTime, endTime time.Time,
) (*ScoreBreakdown, error) {
	// Mock breakdown data
	breakdown := &ScoreBreakdown{
		Dimension: dimension,
		Values: map[string]BreakdownStats{
			"gpt-4": {
				Count:  100,
				Mean:   0.85,
				StdDev: 0.12,
			},
			"gpt-3.5-turbo": {
				Count:  150,
				Mean:   0.72,
				StdDev: 0.18,
			},
			"claude-3-sonnet": {
				Count:  80,
				Mean:   0.88,
				StdDev: 0.10,
			},
		},
	}

	return breakdown, nil
}

// CalculateCohenKappa calculates inter-annotator agreement
func (s *ScoreAnalyticsService) CalculateCohenKappa(
	ctx context.Context,
	projectID uuid.UUID,
	scoreName string,
	annotator1, annotator2 uuid.UUID,
	startTime, endTime time.Time,
) (*CohenKappaResult, error) {
	// Mock confusion matrix (for categorical scores)
	confusionMatrix := map[string]map[string]int{
		"positive": {
			"positive": 45,
			"negative": 5,
			"neutral":  3,
		},
		"negative": {
			"positive": 3,
			"negative": 38,
			"neutral":  2,
		},
		"neutral": {
			"positive": 2,
			"negative": 4,
			"neutral":  18,
		},
	}

	// Calculate observed agreement
	total := 0
	agreed := 0
	for cat1, row := range confusionMatrix {
		for cat2, count := range row {
			total += count
			if cat1 == cat2 {
				agreed += count
			}
		}
	}
	observedAgreement := float64(agreed) / float64(total)

	// Calculate expected agreement (by chance)
	categories := []string{"positive", "negative", "neutral"}
	expectedAgreement := 0.0

	for _, cat := range categories {
		// Marginal probabilities
		annotator1Total := 0
		annotator2Total := 0

		for _, row := range confusionMatrix {
			annotator2Total += row[cat]
		}

		for cat2, count := range confusionMatrix[cat] {
			annotator1Total += count
			_ = cat2 // avoid unused variable
		}

		p1 := float64(annotator1Total) / float64(total)
		p2 := float64(annotator2Total) / float64(total)
		expectedAgreement += p1 * p2
	}

	// Cohen's Kappa
	kappa := (observedAgreement - expectedAgreement) / (1 - expectedAgreement)

	// Interpretation
	interpretation := interpretKappa(kappa)

	return &CohenKappaResult{
		Kappa:           kappa,
		Interpretation:  interpretation,
		Agreement:       observedAgreement,
		ChanceAgreement: expectedAgreement,
		ConfusionMatrix: confusionMatrix,
	}, nil
}

// CalculateF1Score calculates F1 score and related metrics
func (s *ScoreAnalyticsService) CalculateF1Score(
	ctx context.Context,
	projectID uuid.UUID,
	scoreName string,
	threshold float64, // For converting continuous scores to binary
	groundTruthSource string,
	startTime, endTime time.Time,
) (*F1ScoreResult, error) {
	// Mock classification results
	tp := 85  // True Positives
	tn := 90  // True Negatives
	fp := 10  // False Positives
	fn := 15  // False Negatives

	precision := float64(tp) / float64(tp+fp)
	recall := float64(tp) / float64(tp+fn)
	f1 := 2 * (precision * recall) / (precision + recall)
	accuracy := float64(tp+tn) / float64(tp+tn+fp+fn)

	return &F1ScoreResult{
		F1Score:   f1,
		Precision: precision,
		Recall:    recall,
		Accuracy:  accuracy,
		TP:        tp,
		TN:        tn,
		FP:        fp,
		FN:        fn,
	}, nil
}

// GetScoreTrend calculates score trends over time
func (s *ScoreAnalyticsService) GetScoreTrend(
	ctx context.Context,
	projectID uuid.UUID,
	scoreName string,
	startTime, endTime time.Time,
	interval time.Duration,
) (*ScoreTrend, error) {
	// Generate mock trend data
	datapoints := []TrendDatapoint{}

	current := startTime
	for current.Before(endTime) {
		// Simulate increasing trend with some noise
		daysSinceStart := current.Sub(startTime).Hours() / 24
		baseMean := 0.7 + (daysSinceStart * 0.01)

		datapoints = append(datapoints, TrendDatapoint{
			Timestamp: current,
			Mean:      baseMean,
			Count:     20 + int(daysSinceStart),
		})

		current = current.Add(interval)
	}

	// Calculate trend direction
	trend := "stable"
	changeRate := 0.0

	if len(datapoints) >= 2 {
		firstMean := datapoints[0].Mean
		lastMean := datapoints[len(datapoints)-1].Mean
		changeRate = (lastMean - firstMean) / firstMean * 100

		if changeRate > 5 {
			trend = "increasing"
		} else if changeRate < -5 {
			trend = "decreasing"
		}
	}

	return &ScoreTrend{
		ScoreName:  scoreName,
		Datapoints: datapoints,
		Trend:      trend,
		ChangeRate: changeRate,
	}, nil
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func calculateMean(values []float64) float64 {
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func calculateStdDev(values []float64, mean float64) float64 {
	variance := 0.0
	for _, v := range values {
		variance += math.Pow(v-mean, 2)
	}
	variance /= float64(len(values))
	return math.Sqrt(variance)
}

func calculatePercentile(sorted []float64, percentile float64) float64 {
	if len(sorted) == 0 {
		return 0
	}

	rank := percentile / 100.0 * float64(len(sorted)-1)
	lowerIndex := int(math.Floor(rank))
	upperIndex := int(math.Ceil(rank))

	if lowerIndex == upperIndex {
		return sorted[lowerIndex]
	}

	// Linear interpolation
	fraction := rank - float64(lowerIndex)
	return sorted[lowerIndex]*(1-fraction) + sorted[upperIndex]*fraction
}

func createHistogram(values []float64, min, max float64, numBuckets int) []HistogramBucket {
	buckets := make([]HistogramBucket, numBuckets)
	bucketSize := (max - min) / float64(numBuckets)

	// Initialize buckets
	for i := 0; i < numBuckets; i++ {
		buckets[i] = HistogramBucket{
			Min:   min + float64(i)*bucketSize,
			Max:   min + float64(i+1)*bucketSize,
			Count: 0,
		}
	}

	// Count values in each bucket
	for _, v := range values {
		bucketIndex := int((v - min) / bucketSize)
		if bucketIndex >= numBuckets {
			bucketIndex = numBuckets - 1
		}
		if bucketIndex < 0 {
			bucketIndex = 0
		}
		buckets[bucketIndex].Count++
	}

	return buckets
}

func calculatePearsonCorrelation(x, y []float64) float64 {
	n := len(x)
	if n == 0 {
		return 0
	}

	meanX := calculateMean(x)
	meanY := calculateMean(y)

	numerator := 0.0
	denomX := 0.0
	denomY := 0.0

	for i := 0; i < n; i++ {
		dx := x[i] - meanX
		dy := y[i] - meanY
		numerator += dx * dy
		denomX += dx * dx
		denomY += dy * dy
	}

	if denomX == 0 || denomY == 0 {
		return 0
	}

	return numerator / math.Sqrt(denomX*denomY)
}

func calculateSpearmanCorrelation(x, y []float64) float64 {
	// Convert to ranks
	ranksX := rank(x)
	ranksY := rank(y)

	// Calculate Pearson correlation on ranks
	return calculatePearsonCorrelation(ranksX, ranksY)
}

func rank(values []float64) []float64 {
	type indexedValue struct {
		value float64
		index int
	}

	indexed := make([]indexedValue, len(values))
	for i, v := range values {
		indexed[i] = indexedValue{value: v, index: i}
	}

	// Sort by value
	sort.Slice(indexed, func(i, j int) bool {
		return indexed[i].value < indexed[j].value
	})

	// Assign ranks
	ranks := make([]float64, len(values))
	for i, iv := range indexed {
		ranks[iv.index] = float64(i + 1)
	}

	return ranks
}

func interpretKappa(kappa float64) string {
	if kappa < 0 {
		return "Poor (worse than chance)"
	} else if kappa < 0.20 {
		return "Slight agreement"
	} else if kappa < 0.40 {
		return "Fair agreement"
	} else if kappa < 0.60 {
		return "Moderate agreement"
	} else if kappa < 0.80 {
		return "Substantial agreement"
	} else {
		return "Almost perfect agreement"
	}
}
