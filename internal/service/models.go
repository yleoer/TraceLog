package service

type Link struct {
	Title string `json:"title"`
	URL   string `json:"url"`
	Type  string `json:"type"`
}

type Issue struct {
	ID           int64    `json:"id"`
	JiraKey      string   `json:"jira_key"`
	Title        string   `json:"title"`
	Status       string   `json:"status"`
	Priority     string   `json:"priority"`
	Tags         []string `json:"tags"`
	SummaryMD    string   `json:"summary_md"`
	BackgroundMD string   `json:"background_md"`
	AnalysisMD   string   `json:"analysis_md"`
	SolutionMD   string   `json:"solution_md"`
	ActionsMD    string   `json:"actions_md"`
	ResultMD     string   `json:"result_md"`
	TodoMD       string   `json:"todo_md"`
	Links        []Link   `json:"links"`
	StartedAt    string   `json:"started_at"`
	CompletedAt  string   `json:"completed_at"`
	CreatedAt    string   `json:"created_at"`
	UpdatedAt    string   `json:"updated_at"`
}

type IssueTodo struct {
	ID        int64  `json:"id"`
	IssueID   int64  `json:"issue_id"`
	JiraKey   string `json:"jira_key,omitempty"`
	Content   string `json:"content"`
	DueAt     string `json:"due_at"`
	Done      bool   `json:"done"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type IssueEvent struct {
	ID         int64  `json:"id"`
	IssueID    int64  `json:"issue_id"`
	EventType  string `json:"event_type"`
	ContentMD  string `json:"content_md"`
	HappenedAt string `json:"happened_at"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

type TempTaskEvent struct {
	ID         int64  `json:"id"`
	TempTaskID int64  `json:"temp_task_id"`
	EventType  string `json:"event_type"`
	ContentMD  string `json:"content_md"`
	HappenedAt string `json:"happened_at"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

type TempTask struct {
	ID               int64    `json:"id"`
	Title            string   `json:"title"`
	Source           string   `json:"source"`
	Status           string   `json:"status"`
	Priority         string   `json:"priority"`
	Tags             []string `json:"tags"`
	ContentMD        string   `json:"content_md"`
	StartedAt        string   `json:"started_at"`
	CompletedAt      string   `json:"completed_at"`
	ConvertedToJira  bool     `json:"converted_to_jira"`
	ConvertedJiraKey string   `json:"converted_jira_key"`
	CreatedAt        string   `json:"created_at"`
	UpdatedAt        string   `json:"updated_at"`
}

type WeeklyLog struct {
	ID         int64  `json:"id"`
	Week       string `json:"week"`
	SummaryMD  string `json:"summary_md"`
	NextPlanMD string `json:"next_plan_md"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

type WeekBounds struct {
	FirstWeek   string `json:"first_week"`
	CurrentWeek string `json:"current_week"`
}

type DayComment struct {
	Source     string `json:"source"`
	EventID    int64  `json:"event_id"`
	EventType  string `json:"event_type"`
	ContentMD  string `json:"content_md"`
	HappenedAt string `json:"happened_at"`
	RefKey     string `json:"ref_key"`
	RefID      int64  `json:"ref_id"`
	RefTitle   string `json:"ref_title"`
	URL        string `json:"url"`
}

type DayActivity struct {
	Source    string       `json:"source"`
	RefID     int64        `json:"ref_id"`
	RefKey    string       `json:"ref_key"`
	RefTitle  string       `json:"ref_title"`
	URL       string       `json:"url"`
	StartedAt string       `json:"started_at"`
	Comments  []DayComment `json:"comments"`
}

type DayEntry struct {
	ID        int64  `json:"id"`
	Date      string `json:"date"`
	ContentMD string `json:"content_md"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type DayWork struct {
	Date       string        `json:"date"`
	Weekday    string        `json:"weekday"`
	Comments   []DayComment  `json:"comments"`
	Activities []DayActivity `json:"activities"`
	Entries    []DayEntry    `json:"entries"`
}

type WeekView struct {
	Log       WeeklyLog    `json:"log"`
	Issues    []Issue      `json:"issues"`
	Events    []IssueEvent `json:"events"`
	TempTasks []TempTask   `json:"temp_tasks"`
	Todos     []IssueTodo  `json:"todos"`
	Done      []string     `json:"done"`
	Active    []string     `json:"active"`
	Days      []DayWork    `json:"days"`
}

type SearchResult struct {
	Type      string `json:"type"`
	ID        string `json:"id"`
	Title     string `json:"title"`
	Snippet   string `json:"snippet"`
	URL       string `json:"url"`
	UpdatedAt string `json:"updated_at"`
}

type Dashboard struct {
	RecentIssues []Issue     `json:"recent_issues"`
	ActiveIssues []Issue     `json:"active_issues"`
	TempTasks    []TempTask  `json:"temp_tasks"`
	Todos        []IssueTodo `json:"todos"`
	Week         WeekView    `json:"week"`
}

type TodayWorkflow struct {
	Date        string      `json:"date"`
	Issues      []Issue     `json:"issues"`
	TempTasks   []TempTask  `json:"temp_tasks"`
	Todos       []IssueTodo `json:"todos"`
	Done        []string    `json:"done"`
	Active      []string    `json:"active"`
	WeeklyDraft string      `json:"weekly_draft"`
	Day         DayWork     `json:"day"`
}

type AppSettings struct {
	Jira     JiraSettings     `json:"jira"`
	Tempo    TempoSettings    `json:"tempo"`
	AI       AISettings       `json:"ai"`
	OpenAI   ProviderSettings `json:"openai"`
	DeepSeek ProviderSettings `json:"deepseek"`
	Prompts  PromptSettings   `json:"prompts"`
}

type JiraSettings struct {
	BaseURL     string `json:"base_url"`
	Email       string `json:"email"`
	APIToken    string `json:"api_token,omitempty"`
	HasAPIToken bool   `json:"has_api_token"`
}

type TempoSettings struct {
	BaseURL         string `json:"base_url"`
	APIToken        string `json:"api_token,omitempty"`
	HasAPIToken     bool   `json:"has_api_token"`
	AuthorAccountID string `json:"author_account_id"`
}

type AISettings struct {
	Provider string `json:"provider"`
}

type ProviderSettings struct {
	BaseURL   string `json:"base_url"`
	Model     string `json:"model"`
	APIKey    string `json:"api_key,omitempty"`
	HasAPIKey bool   `json:"has_api_key"`
}

type PromptSettings struct {
	IssueSummary  string `json:"issue_summary"`
	WeeklySummary string `json:"weekly_summary"`
}

type IssueSummaryResponse struct {
	Summary string `json:"summary"`
	Issue   Issue  `json:"issue"`
}

type UploadedImage struct {
	URL         string `json:"url"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
}

type UploadedImageData struct {
	URL     string `json:"url"`
	DataURL string `json:"data_url"`
}

type UploadedImageCleanup struct {
	Scanned    int   `json:"scanned"`
	Deleted    int   `json:"deleted"`
	Kept       int   `json:"kept"`
	Failed     int   `json:"failed"`
	FreedBytes int64 `json:"freed_bytes"`
}

type IssueFilter struct {
	Query  string
	Status string
	Tag    string
	Limit  int
	Offset int
	All    bool
}

type TempTaskFilter struct {
	Query  string
	Status string
	Tag    string
	Limit  int
	Offset int
	All    bool
}

type TimeWorkItem struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

type LogTimeRequest struct {
	WorkItemKey string `json:"work_item_key"`
	Description string `json:"description"`
	Hours       int    `json:"hours"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
}

type LogTimeResult struct {
	WorkItemKey string         `json:"work_item_key"`
	Description string         `json:"description"`
	Hours       int            `json:"hours"`
	StartDate   string         `json:"start_date"`
	EndDate     string         `json:"end_date"`
	Total       int            `json:"total"`
	Successful  int            `json:"successful"`
	Failed      int            `json:"failed"`
	Entries     []LogTimeEntry `json:"entries"`
}

type LogTimeEntry struct {
	Date           string `json:"date"`
	TempoWorklogID int64  `json:"tempo_worklog_id"`
	Self           string `json:"self"`
	StartTime      string `json:"start_time"`
	EndTime        string `json:"end_time"`
	Error          string `json:"error,omitempty"`
}

type TimeWeekView struct {
	Week       string         `json:"week"`
	StartDate  string         `json:"start_date"`
	EndDate    string         `json:"end_date"`
	Worklogs   []TimeWorklog  `json:"worklogs"`
	Days       []TimeDay      `json:"days"`
	TotalHours float64        `json:"total_hours"`
	WorkItems  []TimeWorkItem `json:"work_items"`
}

type TimeDay struct {
	Date       string        `json:"date"`
	Weekday    string        `json:"weekday"`
	Worklogs   []TimeWorklog `json:"worklogs"`
	TotalHours float64       `json:"total_hours"`
}

type TimeWorklog struct {
	TempoWorklogID   int64   `json:"tempo_worklog_id"`
	WorkItemKey      string  `json:"work_item_key"`
	WorkItemLabel    string  `json:"work_item_label"`
	Description      string  `json:"description"`
	StartDate        string  `json:"start_date"`
	StartTime        string  `json:"start_time"`
	EndTime          string  `json:"end_time"`
	TimeSpentSeconds int64   `json:"time_spent_seconds"`
	Hours            float64 `json:"hours"`
	Self             string  `json:"self"`
}

type TimeCacheRange struct {
	AccountID   string `json:"account_id"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
	RefreshedAt string `json:"refreshed_at"`
}
