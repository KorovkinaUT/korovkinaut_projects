package sender

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	appsender "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/sender"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/config"
	bothttp "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http/bot"
)

func NewMessageSender(
	transport string,
	baseURL string,
	httpClient *http.Client,
	kafkaCfg *config.KafkaConfig,
	retryCfg *config.RetryConfig,
	cbCfg *config.CircuitBreakerConfig,
	logger *slog.Logger,
) (appsender.MessageSender, error) {
	switch strings.ToUpper(transport) {
	case "HTTP":
		httpSender := NewBotHTTPSender(bothttp.NewClient(baseURL, httpClient, retryCfg, cbCfg))
		kafkaSender := NewBotKafkaSender(kafkaCfg.Brokers, kafkaCfg.ProcessedUpdatesTopic)
		return NewBotFallbackSender(httpSender, kafkaSender, logger), nil
	case "KAFKA":
		return NewBotKafkaSender(kafkaCfg.Brokers, kafkaCfg.ProcessedUpdatesTopic), nil
	default:
		return nil, fmt.Errorf("unknown updates transport: %q", transport)
	}
}
