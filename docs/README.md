# Documentation

Ticker Lab project documentation.

## Index

| Document                                                                      | Description                                                              |
| ----------------------------------------------------------------------------- | ------------------------------------------------------------------------ |
| [architecture.md](architecture.md)                                            | Hexagonal architecture, polyglot services, data flow                     |
| [api.md](api.md)                                                              | REST API endpoints (Node + Go), response formats, data sources           |
| [runbook.md](runbook.md)                                                      | Local dev, operations, troubleshooting                                   |
| [changelog.md](changelog.md)                                                  | Reverse-chronological change log                                         |
| [future-providers.md](future-providers.md)                                    | Roadmap of data providers to integrate                                   |
| [tech-debt-analysis.md](tech-debt-analysis.md)                                | Pre-Phase 12 tech debt audit + test expansion plan + Google AIP audit    |
| [api-design-standards.md](api-design-standards.md)                            | Binding API design rules adopted from Google AIP (with deviations)       |
| [comparison-providers-research.md](comparison-providers-research.md)          | API research: insurance, investment products, utilities                  |
| [esios-integration.md](esios-integration.md)                                  | Integration spec: REE ESIOS (Spanish electricity)                        |
| [cnmv-integration.md](cnmv-integration.md)                                    | Integration spec: CNMV public files (Spanish funds NAV)                  |
| [bde-integration.md](bde-integration.md)                                      | Integration spec: Banco de España (Spanish rates)                        |
| [macro-indicators-integration.md](macro-indicators-integration.md)            | Integration spec: FRED + ECB (macro indicators)                          |
| [ratehawk-integration.md](ratehawk-integration.md)                            | Integration spec: RateHawk (deprecated/abandoned)                        |
| [future-features.md](future-features.md)                                      | Feature backlog                                                          |
| [decisions/README.md](decisions/README.md)                                    | Architecture Decision Records (ADRs) index                               |

## ADRs

| #                                              | Title                                                | Status   |
| ---------------------------------------------- | ---------------------------------------------------- | -------- |
| [001](decisions/001-tech-stack.md)             | Technology Stack                                     | Accepted |
| [002](decisions/002-frontend-ssr.md)           | Fastify SSR Instead of React SPA                     | Accepted |
| [003](decisions/003-hosting-strategy.md)       | Hosting Strategy — Hetzner Self-Hosted Multi-Project | Proposed |
