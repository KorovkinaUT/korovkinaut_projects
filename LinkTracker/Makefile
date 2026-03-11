# Loads environment variables from .env for reading them via os.Getenv
# and run telegram bot and scrapper
.PHONY: bot scrapper run

bot:
	@set -a; . ./.env; set +a; go run ./cmd/bot

scrapper:
	@set -a; . ./.env; set +a; go run ./cmd/scrapper

run:
	@set -a; . ./.env; set +a; \
	go run ./cmd/bot & \
	BOT_PID=$$!; \
	go run ./cmd/scrapper & \
	SCRAPPER_PID=$$!; \
	trap "kill $$BOT_PID $$SCRAPPER_PID" INT TERM; \
	wait