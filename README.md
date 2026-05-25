# BTL-2526II_ELT3098_1

## Status
29/04/2026: Initial project skeleton and planning documents are ready.

## Suggested Tech Stack

| Area | Technology | Suggested Version | Official Docs |
|---|---|---|---|
| Orbit calculation | Python | 3.11.x | https://docs.python.org/3/ |
| Scientific computing | NumPy | 2.x | https://numpy.org/doc/ |
| Orbit propagation | SGP4 | latest stable | https://pypi.org/project/sgp4/ |
| Astronomy/time coords | Astropy | 6.x | https://docs.astropy.org/en/stable/ |
| Orbit analysis | Poliastro | latest stable compatible with chosen Python | https://docs.poliastro.space/ |
| Notebook simulation | JupyterLab | 4.x | https://jupyterlab.readthedocs.io/en/stable/ |
| Core backend | Go | 1.24.x | https://go.dev/doc/ |
| Desktop app shell | Tauri | v2.x | https://v2.tauri.app/start/ |
| Desktop backend | Rust | stable (latest) | https://www.rust-lang.org/learn |
| Client frontend | Svelte | 5.x | https://svelte.dev/docs |
| Web admin framework | SvelteKit | 2.x | https://kit.svelte.dev/docs |
| Styling | Tailwind CSS | 3.x or 4.x (match SvelteKit plugin support) | https://tailwindcss.com/docs |
| Package manager (JS) | npm | 10.x+ | https://docs.npmjs.com/ |

## Environment Notes

Prefer separate environments per module:  
1. Python venv for `orbit_calc/`  
2. Go modules for `core_network/`  
3. Rust toolchain for `client_app/src-tauri/`  
4. Node/npm workspace for `client_app/src/` and `web_admin/`  

## Project Structure

```
BTL-2526II_ELT3098_1/
├── .gitignore                      Standard exclusions for Python/Go/Node/Rust projects
├── agents.md                       Team roles, tasks, dependencies, success criteria
├── README.md                       
│
├── docs/
│   ├── architecture.md             System design, data flows, tech stack, deployment
│   └── BaiTapNhom.md               Original project specification
│
├── orbit_calc/                     Orbit & Coverage (P1)
│   └── remediation_list.md         Coverage calculator, traffic model, simulation tasks
│
├── core_network/                   Gateway & Handover (P1 + P2)
│   └── remediation_list.md         Gateway nodes, handover manager, provisioning, billing
│
├── client_app/                     End-User Router (P1 + P2)
│   ├── src-tauri/                  Antenna tracker, geo-reporter (Rust backend)
│   ├── src/                        Signal dashboard, data usage (Svelte frontend)
│   └── remediation_list.md         UI/UX components, Tauri setup, testing tasks
│
└── web_admin/                      ISP Admin Dashboard (P1 + P2)
    ├── src/routes/
    │   ├── monitoring/             Detailed monitoring page (P1)
    │   └── security/               Device registry & billing (P2)
    └── remediation_list.md         SvelteKit setup, API integration, admin workflows

```