package match

type Verdict struct {
	Status     string
	Confidence float64
	Fields     []FieldResult
}

func Aggregate(fields []FieldResult) Verdict {
	status := "consistent"
	if len(fields) == 0 {
		return Verdict{Status: status, Confidence: 1, Fields: fields}
	}

	total := 0.0
	for _, field := range fields {
		if !field.Pass {
			status = "flagged"
		}
		total += field.Score
	}

	return Verdict{
		Status:     status,
		Confidence: total / float64(len(fields)),
		Fields:     fields,
	}
}
