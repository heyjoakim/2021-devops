# Weekly Log

## Week 01

- [X] Add version control
    - [Repo](https://github.com/heyjoakim/devops-21)
- [X] Try to develop a high-level understanding of ITU-MiniTwit.
- [X] Migrate ITU-MiniTwit to run on a modern computer running Linux
    - [X] Get python to run
    - [X] Install deps
    - [X] Recompile flag_tool
    - [X] Install SQLite browser
    - [x] Run 2to3 to convert py2 to py3
    - [x] `shellcheck` and fix `control.sh`


- [X] Share Work on GitHub
    - [Repo](https://github.com/heyjoakim/devops-21)
- [X] Prep for next week 
    - [X] Discussed branching strategy, explained below in the notes section

## Notes
We meet on Mondays from 10.00 - X.X.X.X (Super agile here!!)

### This is our branching strategy
PR
Develop -> Feature

Branch out from develop into a feature / bug and then create a PR to merge back into develop. From develop releases are pushed to production (maybe one test environment?).


## Week 02

- [X] Choose language and technology for refactoring
    - [X] And why
- [X] Choose branching strategy
- [X] Refactor
- [X] Commitment guidelines?
- [X] Implement API for simulator

### Choose language and technology

|Lang/Dev|Pros|Cons|
|---|---|---|
|Go/Gorilla   |Fast compared to other suggested frameworks [[1]](https://github.com/the-benchmarker/web-frameworks), fullstack   |Setting up env can be tricky   |
|C#/ASP.NET/Razor/Blazor   |Scalable, plenty of resources, fullstack  |Somewhat heavy framework [[1]](https://github.com/the-benchmarker/web-frameworks), not easy to make 1:1 mapping, due to different structure, Still early life  |
|JS/Angualr   |Strong community, fullstack   |Not suited for 1:1 app, Not statically type |
|JS/Vue   |Easy to get started with, lightweight   |Not suited for 1:1 app, Not statically type, Needs separate backend |

We have chosen Go as we believe this is well suited for such task and is fast compared to other frameworks.

### Choose a branching strategy
We are discussing advantages and disadvantages between a Git Flow and Topic/Feature workflow strategy.

|Strategy | Pros | Cons
|---|---|---|
|Git flow| Separate releases, more "controlled", more suited for weekly release | "more" work |
|Feature workflow | Continous development, cleaner Git history, Simple, Faster deploys | Need more internal communication |

We have chosen to go with a modified Git Flow strategy as we believe this is more suited for our weekly releases. We have decided to omit the release branch, since we think that it would create unnecessary overhead compared to the size of the project. Our _development_-branch will do tests once CI/CD i setup. 

![](https://i.imgur.com/ea6o39W.png)

The branch structure will therefore be as following :

- `develop` All new feature branches must check out from here into feature branches and merged back into develop. The contents of the development branch would usually reflect what is deployed to the test environment.
- `main` The production branch reflects the current deployment in production. The production branch is merged with the develop branch every time a new version deployed to production.
- `feature/{feature-name}` New features are developed on feature branches following the *feature / feature name branch* structure.
-  `hotfix\{hotfix-name}` New hot fixes are developed on separate hot fix branches following the *hotfix / hotfix branch name*

## Week 03 Virtalization

- [X] Complete implementing an API for the simulator 
- [X] Continue refactoring 
  - [X] Introduce a DB abstraction layer
  - [X] Arguments for choice of ORM framework and chosen DBMS
  - [ ] Rebase and deploy
  - [ ] Provide good arguments for choice of virtualization techniques and deployment targets
- [ ] Log dependencies

#### Release and deploy
Azure as cloud provider with docker!

#### ORM Tool
We decided to use [GORM](https://github.com/go-gorm/gorm) as it is one of the most widely used ORMs for Golang (reference:https://github.com/go-gorm/gorm) and also after further research we found it to be the most well-documented.
We also discussed switching to PostgreSQL as a datasource, but decided to postpone that to a later stage, as the ORM abstraction will give us the flexibility to change data storages. 

## Choice of ORM
So far, the application had been constructing it's own SQL statements, and executing them as prepared statements, using SQLite3. However, we need to find a way to best prepare ourselves and minitwit for any changes that may have to be done. 

Gorm makes it possible for us to use the golang structs that we already have been working with, in such a way that we can save our objects directly to the database, thereby also having a more explicit struct-strategy in our code.
We also expect that later in the course, it might become necessary to do some refractoring of the database, which is easier with the code-first workflow of Gorm. In that respect, we expect to be able to more dynamically manipulate our database codefirst. Creating primary keys, columns and rows can all be manipulated and created code first. 
We also hope to be able to get rid of some repetetive boilerplate SQL, and thus make the code more readable, to the non SQL initiated developer.

Another positive benefit could be that changing to another dbms, could require less work in terms of rewriting code, thus improving modifiability.

## Staying with SQLite
For now we have decided to keep SQLite3 as SQL database engine as the database is still of a smaller size and the write volumes are low. 

TO BE DELETED