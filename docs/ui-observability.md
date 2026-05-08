# Lucifer TUI: Real-Time Chaos Observability

The Lucifer Terminal User Interface (TUI) is a high-fidelity, event-driven observability dashboard designed for **Chaos Forensics**. Unlike traditional monitoring tools that rely on slow polling (like CloudWatch), the Lucifer TUI provides sub-second visibility into infrastructure spikes during active attack simulations.

## Dashboard Architecture

The dashboard is organized into a modular grid that prioritizes real-time impact visualization and security audit status.

### 1. Sidebar Control Plane (Left)
*   **Active Agents (Top Left)**: A list of discovered chaos agents. Use the arrow keys to select an agent to filter the dashboard's forensic data.
*   **Experiment History (Bottom Left)**: A historical record of all experiments (CPU stress, latency injection, etc.) performed on the selected agent.

### 2. Forensic Visualization Grid (Center/Right)
The core of the dashboard is a set of real-time plots focused on the **Three Pillars of Chaos**:
*   **CPU Impact (%)**: Visualizes the processing overhead during stress tests.
*   **Memory Impact (MB)**: Tracks memory consumption and leakage during heap-stress experiments.
*   **Latency Impact (ms)**: Monitors network degradation during latency injection.


### 3. Intelligence & Health Panels (Top/Bottom)
*   **System Health**: Displays averages for CPU, Memory, and Latency.
*   **Live Activity Feed**: A scrolling real-time log of agent heartbeats and experiment state changes.
*   **Experiment Progress Gauge**: A dedicated timer that visualizes the remaining duration of an active chaos task.
*   **VAPT Findings**: A summary of the VAPT results for the selected agent.

### 4. How to run the UI
* Start the orchestrator, `go run chaos-engineering/orchestrator/main.go`.
* Start the datamodel, `go run datamodel/cmd/server/main.go`.
* Start the TUI, `go run UI/tui/tui.go`.
