# Aj Bell Touch Test

This project uses go lang & fiber to solve the Aj bell tech touch tech test.


## Installation

### Prerequisites

Make sure you have the following installed on your system:

- Go (version 1.19.1)
- Make
- Postgres

### Steps

1. Update config.yml with your database configuration
2. Install dependencies

   ```bash
   make deps
3. Build the project:
   ``` bash
   make build
   
4. To run the unit tests and integration tests:
   ``` bash
   make test
   
If you would like to see the test coverage:
   ``` make test_coverage  ```

### Endpoints

1. GET - /api/v1/deposit/:id -> returns the deposit and the allocations
2. POST - /api/v1/deposit -> Creates a deposit 
3. POST - /api/v1/deposit/:id/receipt

