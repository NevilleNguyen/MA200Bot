# MA200Bot

Used to track when price of multiple pairs hit the MA200, notify to telegram.

## Configuration
There are two configurations:
- In `env` folder: contains public configuration
- In `.env` file: contains private configuration

So for our app:
- If we want to track specific pairs, set field `symbols` in file `mainnet.json` in `env` folder. Or if we want to exclude pairs, set field `excluded_symbols`.
- If we want to change timeframes, set field `timeframes`
- Create `.env` file with variable names like in `env_example` file.

## Run
Execute command: `go run main.go`