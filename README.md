# DBInsight - MySQL Proxy and Load Balancer

DBInsight is a high-performance, scalable, and secure proxy and load balancer for MySQL databases. It sits between your applications and your MySQL servers, providing connection pooling, query routing, basic query inspection, and more.  It's designed to improve database performance, availability, and security.

## Features

*   **Connection Pooling:** Efficiently manages connections to your MySQL servers, reducing connection overhead and improving application performance.
*   **Read-Only Query Routing:** Routes read-only queries to dedicated read replicas, offloading read traffic from your primary database server.
*   **Basic Query Inspection:** Logs query execution time and allows for basic analysis of queries.  (More advanced query analysis features are planned.)
*   **High Availability:**  Can be configured to route traffic to multiple MySQL servers, providing redundancy and failover capabilities.
*   **Scalability:** Designed to handle a large number of connections and queries.
*   **Security:** Helps protect your database servers by acting as an intermediary and providing a layer of security.
*   **Easy Deployment:**  Deployable as a Docker container or as a native binary.
*   **Metrics and Monitoring:** Integrates with Prometheus and Grafana for monitoring key performance indicators.
*   **Alerting:**  (Planned)  Will provide alerting capabilities for critical events.

## Getting Started

### Prerequisites

*   Go 1.20 or later
*   Docker (for containerized deployment)
*   Packer (for building QEMU images)
*   MySQL Servers

### Installation

#### From Source

1.  Clone the repository:

    ```bash
    git clone https://github.com/jbhall78/dbinsight.git
    ```

2.  Navigate to the project directory:

    ```bash
    cd dbinsight
    ```

3.  Build the proxy:

    ```bash
    make
    ```

#### Using Docker

1.  Clone the repository (as above).

2.  Build the Docker image:

    ```bash
    docker build -t dbinsight-proxy .
    ```
    or

    ```bash
    make docker-build
    ```

### Configuration

The proxy is configured using a YAML configuration file.  The configuration file (`data/config/config.yaml`) is provided in the repository.  You'll need to configure the following:

*   `backend_primary_*`:  Variables for your MySQL server addresses and credentials.
*   `backend_replicas`: A list of read replica server addresses and credentials (if using read/write splitting).
*   `listen_address`: The address and port the proxy listens on.
*   `log_level`: The log level (e.g., debug, info, warn, error).

### Running the Proxy

#### From Source

```bash
./dbinsight-proxy
