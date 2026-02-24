# TODO

- [ ] Add auto-sync capability for the repo/ws/portfolio info and add a sync button - The auto-sync should be enabled by default with like 1 minute period -> user can have a dropdown they can choose from to change the auto-sync period from like 30 seconds to 5 minutes - this is only front-end side auto-sync --- This can obviously only work with WASM

## Done

- [x] Implement /portfolio as new root route for workspaces a portfolio is a collection of workspaces -> this allow creating hierarchy of workspaces for larger projects and/or helping grouping workspaces logically
- [x] In /workspaces display most recently modified workspaces sorted from most recent modified date to least recent modified
- [x] Refactor UI using <https://m3.material.io/> -> Use the frontend-design-material skill in /home/alexandremahdhaoui/go/src/github.com/alexandremahdhaoui/skills
- [x] Portfolios should also have a forge-portfolio.yaml that describes the portfolio purpose and the and points to .forge-ai portfolio tracker/plan
- [x] Workspaces should have some documentation to describe what they are meant for and this description should be included into the forge-ui
  - We use a forge-workspace.yaml file that defines the workspace structure, has a description etc...
  - (this would be then used with a forge-ws MCP server maybe to help working with workspaces)
  - The forge-workspace.yaml could track the meta-plans in the workspace (like path to it) and the progress to meta-plans could then be reported in the forge-ui -> the .ai plans or .forge-ai/plan (plans) in the repo (not workspace) could then be used to also track on a per repo level
- [x] Implement META_ORCHESTRATOR mode for working with forge workspaces: the META_ORCHESTRATOR creates an overall plan that will be used to track smaller piece of work in each repo of the workspace. The work will be tracked in .forge-ai/ directories -> the .forge-ai/ directories from the workspace contain a .forge-ai/meta-plan -> that explains features and orchestration of task across multiple repositories and multiples stages with milestones and checkpoints where overall testing must be done etc...
  - You must create a Claude Skill for META_ORCHESTRATOR mode and also a Skill for ORCHESTRATOR mode
  - Update forge-ui to ensure it's using the forge-workspace.yaml
- [x] Remove the color grey from the light UI
- [x] Create the CONTRIBUTING, DESIGN.md and README.md for this repo
