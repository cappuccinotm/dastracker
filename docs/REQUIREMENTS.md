## Requirements

#### Glossary
- Tracker - task tracker (e.g. Asana, Jira)
- Driver - rpc server for work with a specific tracker
- Trigger - external event in some tracker
- Job - program routine which runs on a specific trigger
- Action - driver's method which takes variables through 'with' parameter

#### Stakeholders roles
- Yelshat Duskaliyev - CTO, Project manager
- Oybek Kasimov - System analyst, QA engineer

#### User stories
- As a system admin I want to be able to configure access to trackers on startup so that I can choose which trackers 
to synchronize
- As a tracker user I want my tasks to be updated as soon as there are changes in other trackers so that I always have 
the latest version of tasks
- As a tracker user I want to be able to add my Telegram account so that I get notifications on tasks updates
- As a tracker user I want to get visual notification when task synchronization is failed so that I know about pending 
updates

#### Non-functional requirements
- **Performance**: Task synchronization must not take more than 3 seconds for any specific task
- **Performance**: Telegram notifications must be received in no more than 10 seconds
- **Capacity**: The system must operate with up to 100 simultaneous synchronizations
- **Security**: Only system administrator has access to confidential information
- **Data integrity**: System must fully recover after long-term outage or hard shutdown
- **Usability**: System must require minimum configuration and maintenance
