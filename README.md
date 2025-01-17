# A Small Project: Visitor Counter for My Website
![image](https://github.com/user-attachments/assets/2f7490ad-c0b6-4246-9d27-90edc63de2e2)


## Features

- **Increment Visitor Count**: Tracks unique visitors by their IP address and stores the data in Redis with features like Set, HLL, transactions and Lua scripts.
- **Get Visitor Counts**: Retrieves the count of unique visitors for the current and previous month, categorized by country.
- **IP-to-Geo Integration**: Converts IP addresses to geographical information using an external API.
- **Logging**: Asynchronuous logging supported (👷🏼 TODO: provide reports of traffic history)

## Prerequisites

- Go 1.22
- Redis server
- Access to the IP-to-Geo service with a valid API key
- protoc-gen-go-grpc@latest

## Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/visitor-counter-svc.git
   cd visitor-counter-svc
   ```

2. Install dependencies:
   ```bash
   make init
   ```

3. Set up environment variables:
## Environment Variables
- **`IP2LOCATION_API_KEY`**: 
  - **Description**: API key for accessing the IP2Location service, which is used to convert IP addresses into geographical information.
  - **Example**: `123456789abcdef`

- **`PORT`**: 
  - **Description**: The port number on which the application will run.
  - **Example**: `8080`

- **`REDIS_ADDR`**: 
  - **Description**: The address of the Redis server used for storing and retrieving visitor data.
  - **Example**: `localhost:6379`

- **`TARGETED_COUNTRIES`**: 
  - **Description**: A comma-separated list of country codes that the application should specifically target or monitor.
  - **Example**: `US,CA,GB`

Ensure these environment variables are set in your environment or in a `.env` file before running the application.

## Usage

### Running the Service

To start the service, run the following command:
```bash
make run
```


### API Endpoints

### `IncrementVisitorCount`

- **Functionality**: Increments the visitor count for a specified IP address.
- **Process**:
  1. Validates the IP address format.
  2. Checks if the IP is already recorded for the current month in Redis.
  3. Retrieves geographical information for the IP using an external API.
  4. Updates visitor counts in Redis, categorized by country.
  5. Logs the visitor's country information.

### `GetVisitorCounts`

- **Functionality**: Retrieves visitor counts for the current and previous month.
- **Process**:
  1. Identifies the current and previous month.
  2. Collects Redis keys for visitor data.
  3. Executes a Lua script to calculate visitor counts using Redis HyperLogLog.
  4. Returns a map of visitor counts grouped by country code.

### Running the Service

To start the tests, run the following command:
```bash
make test
```

## Logging

The service uses `zap` for logging. Logs are output to the console and can be configured for different log levels.
