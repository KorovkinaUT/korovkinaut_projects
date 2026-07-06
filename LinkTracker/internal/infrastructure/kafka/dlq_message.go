package kafkainfra

type DeadLetterMessage struct {
	OriginalTopic     string `json:"originalTopic"`
	OriginalPartition int    `json:"originalPartition"`
	OriginalOffset    int64  `json:"originalOffset"`
	Key               string `json:"key"`
	Value             string `json:"value"`
	Error             string `json:"error"`
}
