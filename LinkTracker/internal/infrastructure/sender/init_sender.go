package sender

import (
	"fmt"
	"net/http"
	"strings"

	appsender "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/sender"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/config"
	bothttp "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http/bot"
)

func NewMessageSender(
	transport string,
	cfg *config.KafkaConfig,
	botBaseURL string,
	httpClient *http.Client,
) (appsender.MessageSender, error) {
	switch strings.ToUpper(transport) {
	case "HTTP":
		return NewHTTPSender(bothttp.NewClient(botBaseURL, httpClient)), nil
	case "KAFKA":
		return NewKafkaSender(cfg.Brokers, cfg.UpdatesTopic), nil
	default:
		return nil, fmt.Errorf("unknown updates transport: %q", transport)
	}
}
