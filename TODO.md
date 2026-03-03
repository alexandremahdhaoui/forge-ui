# TODO

- [ ] I don't like the repository structure some stuff not under ./internal/
- [ ] Add feature to have from the browser access to a terminal console connected to tmux locally
- [ ] Improve /orchestrator to update in repo/workspace/portfolio what's currently being worked on -> like current context about the changes and what's currently being tackled and what problems we have if any etc -> like surfacing blockers etc and scope changes

## Done

- [x] Add some sort of recap/recommendations for the pages: I want to see repos/workspaces/portfolios that have been left unattended -> i.e.: they have uncommited changes or unpushed commits but the repo/workspace/portfolios haven't been modified recently -> Shows work that have been left unattended
- [x] Add auto-sync capability for the repo/ws/portfolio info and add a sync button - The auto-sync should be enabled by default with like 1 minute period -> user can have a dropdown they can choose from to change the auto-sync period from like 30 seconds to 5 minutes - this is only front-end side auto-sync --- This can obviously only work with WASM
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
