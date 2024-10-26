package helper

type Configuration struct {
	Database   Database
	Collection Collection
	Edge       Edge
	Graph      Graph
}

type Database struct {
	Knowledge string
	Results   string
	Plugins   string
	Config    string
}

type Collection struct {
	JS        string
	PHP       string
	Java      string
	Python    string
	Golang    string
	CPP       string
	Versions  string
	Revisions string
	Sboms     string
	Vulns     string
	CWE       string
	NVD       string

	Patches         string
	Vulnerabilities string
	Licenses        string
	Projects        string
	Analyses        string
	Organizations   string
	Integrations    string
	Analyzers       string
	Policies        string
	Users           string
	Apikeys         string
	Plugins         string
	Services        string
	Osv             string
	Results         string
	Config          string
	Notifications   string

	Apikeys_usage_logs              string
	Email_unsubscribtions           string
	Email_unsubscribtions_actions   string
	Github_repos_cache              string
	Github_repos_cache_meta         string
	Gitlab_repos_cache              string
	Gitlab_repos_cache_meta         string
	Organizations_audit_logs        string
	Social_registration_temp        string
	Users_password_reset_requests   string
	Users_registration_verification string
	Organizations_invitations       string
}

type Edge struct {
	Licenses                   string
	Versions                   string
	Plugins                    string
	Services                   string
	Apikey_user                string
	Organizations_analyzers    string
	Organizations_integrations string
	Organizations_policies     string
	Organizations_projects     string
	Organizations_users        string
	Project_analyses           string
	Results                    string
	User_notifications         string
}

type Graph struct {
	Plugins            string
	Services           string
	Dependencies       string
	Licenses           string
	Results            string
	Projects_analysis  string
	Api_key_user       string
	Org_analyzer       string
	Org_integrations   string
	Org_membership     string
	Org_policies       string
	Org_project        string
	User_notifications string
}

var Config Configuration = Configuration{
	Database: Database{
		Knowledge: "knowledge",
		Results:   "codeclarity",
		Plugins:   "plugins",
		Config:    "config",
	},
	Collection: Collection{
		JS:        "JS",
		PHP:       "PHP",
		Java:      "JAVA",
		Python:    "PYTHON",
		Golang:    "GOLANG",
		CPP:       "CPP",
		Versions:  "VERSIONS",
		Revisions: "REVISIONS",
		Sboms:     "SBOMS",
		Vulns:     "VULNS",
		CWE:       "CWE",
		NVD:       "NVD",

		Patches:         "PATCHES",
		Vulnerabilities: "VULNERABILITIES",
		Licenses:        "LICENSES",
		Projects:        "PROJECTS",
		Analyses:        "ANALYSES",
		Organizations:   "ORGANIZATIONS",
		Integrations:    "INTEGRATIONS",
		Analyzers:       "ANALYZERS",
		Policies:        "POLICIES",
		Users:           "USERS",
		Apikeys:         "API_KEYS",
		Plugins:         "PLUGINS",
		Services:        "SERVICES",
		Osv:             "OSV",
		Results:         "RESULTS",
		Config:          "CONFIG",
		Notifications:   "NOTIFICATIONS",

		Apikeys_usage_logs:              "API_KEYS_USAGE_LOGS",
		Email_unsubscribtions:           "EMAIL_UNSUBSCRIPTIONS",
		Email_unsubscribtions_actions:   "EMAIL_UNSUBSCRIPTIONS_ACTIONS",
		Github_repos_cache:              "GITHUB_REPOS_CACHE",
		Github_repos_cache_meta:         "GITHUB_REPOS_CACHE_META",
		Gitlab_repos_cache:              "GITLAB_REPOS_CACHE",
		Gitlab_repos_cache_meta:         "GITLAB_REPOS_CACHE_META",
		Organizations_audit_logs:        "ORGANIZATIONS_AUDIT_LOGS",
		Social_registration_temp:        "SOCIAL_REGISTRATION_TEMP",
		Users_password_reset_requests:   "USERS_PASSWORD_RESET_REQUESTS",
		Users_registration_verification: "USERS_REGISTRATION_VERIFICATION",
		Organizations_invitations:       "ORGANIZATIONS_INVITATIONS",
	},
	Edge: Edge{
		Licenses:                   "LICENSES_EDGES",
		Versions:                   "VERSIONS_EDGES",
		Plugins:                    "PLUGINS_EDGES",
		Services:                   "SERVICES_EDGES",
		Apikey_user:                "API_KEYS_USER_EDGES",
		Organizations_analyzers:    "ORGANIZATIONS_ANALYZERS_EDGES",
		Organizations_integrations: "ORGANIZATIONS_INTEGRATIONS_EDGES",
		Organizations_policies:     "ORGANIZATIONS_POLICIES_EDGES",
		Organizations_projects:     "ORGANIZATIONS_PROJECTS_EDGES",
		Organizations_users:        "ORGANIZATIONS_USERS_EDGES",
		Project_analyses:           "PROJECT_ANALYSES_EDGES",
		Results:                    "RESULTS_EDGES",
		User_notifications:         "USER_NOTIFICATIONS_EDGES",
	},
	Graph: Graph{
		Plugins:            "PLUGINS_GRAPH",
		Services:           "SERVICES_GRAPH",
		Dependencies:       "DEPENDENCIES_GRAPH",
		Licenses:           "LICENSES_GRAPH",
		Results:            "RESULTS_GRAPH",
		Projects_analysis:  "PROJECT_ANALYSIS_GRAPH",
		Api_key_user:       "API_KEY_USER_GRAPH",
		Org_analyzer:       "ORG_ANALYZER_GRAPH",
		Org_integrations:   "ORG_INTEGRATIONS_GRAPH",
		Org_membership:     "ORG_MEMBERSHIP_GRAPH",
		Org_policies:       "ORG_POLICIES_GRAPH",
		Org_project:        "ORG_PROJECT_GRAPH",
		User_notifications: "USER_NOTIFICATIONS_GRAPH",
	},
}
