export namespace desktop {
	
	export class FileUpload {
	    name: string;
	    data: string;
	
	    static createFrom(source: any = {}) {
	        return new FileUpload(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.data = source["data"];
	    }
	}
	export class SaveResult {
	    path: string;
	    canceled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SaveResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.canceled = source["canceled"];
	    }
	}

}

export namespace service {
	
	export class AISettings {
	    provider: string;
	
	    static createFrom(source: any = {}) {
	        return new AISettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = source["provider"];
	    }
	}
	export class PromptSettings {
	    issue_summary: string;
	    weekly_summary: string;
	
	    static createFrom(source: any = {}) {
	        return new PromptSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.issue_summary = source["issue_summary"];
	        this.weekly_summary = source["weekly_summary"];
	    }
	}
	export class ProviderSettings {
	    base_url: string;
	    model: string;
	    api_key?: string;
	    has_api_key: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ProviderSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.base_url = source["base_url"];
	        this.model = source["model"];
	        this.api_key = source["api_key"];
	        this.has_api_key = source["has_api_key"];
	    }
	}
	export class JiraSettings {
	    base_url: string;
	    email: string;
	    api_token?: string;
	    has_api_token: boolean;
	
	    static createFrom(source: any = {}) {
	        return new JiraSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.base_url = source["base_url"];
	        this.email = source["email"];
	        this.api_token = source["api_token"];
	        this.has_api_token = source["has_api_token"];
	    }
	}
	export class AppSettings {
	    jira: JiraSettings;
	    ai: AISettings;
	    openai: ProviderSettings;
	    deepseek: ProviderSettings;
	    prompts: PromptSettings;
	
	    static createFrom(source: any = {}) {
	        return new AppSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.jira = this.convertValues(source["jira"], JiraSettings);
	        this.ai = this.convertValues(source["ai"], AISettings);
	        this.openai = this.convertValues(source["openai"], ProviderSettings);
	        this.deepseek = this.convertValues(source["deepseek"], ProviderSettings);
	        this.prompts = this.convertValues(source["prompts"], PromptSettings);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DayEntry {
	    id: number;
	    date: string;
	    content_md: string;
	    created_at: string;
	    updated_at: string;
	
	    static createFrom(source: any = {}) {
	        return new DayEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.date = source["date"];
	        this.content_md = source["content_md"];
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	    }
	}
	export class DayComment {
	    source: string;
	    event_id: number;
	    event_type: string;
	    content_md: string;
	    happened_at: string;
	    ref_key: string;
	    ref_id: number;
	    ref_title: string;
	    url: string;
	
	    static createFrom(source: any = {}) {
	        return new DayComment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source = source["source"];
	        this.event_id = source["event_id"];
	        this.event_type = source["event_type"];
	        this.content_md = source["content_md"];
	        this.happened_at = source["happened_at"];
	        this.ref_key = source["ref_key"];
	        this.ref_id = source["ref_id"];
	        this.ref_title = source["ref_title"];
	        this.url = source["url"];
	    }
	}
	export class DayWork {
	    date: string;
	    weekday: string;
	    comments: DayComment[];
	    entries: DayEntry[];
	
	    static createFrom(source: any = {}) {
	        return new DayWork(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.weekday = source["weekday"];
	        this.comments = this.convertValues(source["comments"], DayComment);
	        this.entries = this.convertValues(source["entries"], DayEntry);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class IssueEvent {
	    id: number;
	    issue_id: number;
	    event_type: string;
	    content_md: string;
	    happened_at: string;
	    created_at: string;
	    updated_at: string;
	
	    static createFrom(source: any = {}) {
	        return new IssueEvent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.issue_id = source["issue_id"];
	        this.event_type = source["event_type"];
	        this.content_md = source["content_md"];
	        this.happened_at = source["happened_at"];
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	    }
	}
	export class WeeklyLog {
	    id: number;
	    week: string;
	    summary_md: string;
	    next_plan_md: string;
	    created_at: string;
	    updated_at: string;
	
	    static createFrom(source: any = {}) {
	        return new WeeklyLog(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.week = source["week"];
	        this.summary_md = source["summary_md"];
	        this.next_plan_md = source["next_plan_md"];
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	    }
	}
	export class WeekView {
	    log: WeeklyLog;
	    issues: Issue[];
	    events: IssueEvent[];
	    temp_tasks: TempTask[];
	    todos: IssueTodo[];
	    done: string[];
	    active: string[];
	    days: DayWork[];
	
	    static createFrom(source: any = {}) {
	        return new WeekView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.log = this.convertValues(source["log"], WeeklyLog);
	        this.issues = this.convertValues(source["issues"], Issue);
	        this.events = this.convertValues(source["events"], IssueEvent);
	        this.temp_tasks = this.convertValues(source["temp_tasks"], TempTask);
	        this.todos = this.convertValues(source["todos"], IssueTodo);
	        this.done = source["done"];
	        this.active = source["active"];
	        this.days = this.convertValues(source["days"], DayWork);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class IssueTodo {
	    id: number;
	    issue_id: number;
	    jira_key?: string;
	    content: string;
	    due_at: string;
	    done: boolean;
	    created_at: string;
	    updated_at: string;
	
	    static createFrom(source: any = {}) {
	        return new IssueTodo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.issue_id = source["issue_id"];
	        this.jira_key = source["jira_key"];
	        this.content = source["content"];
	        this.due_at = source["due_at"];
	        this.done = source["done"];
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	    }
	}
	export class TempTask {
	    id: number;
	    title: string;
	    source: string;
	    status: string;
	    priority: string;
	    tags: string[];
	    content_md: string;
	    result_md: string;
	    started_at: string;
	    completed_at: string;
	    converted_to_jira: boolean;
	    converted_jira_key: string;
	    created_at: string;
	    updated_at: string;
	
	    static createFrom(source: any = {}) {
	        return new TempTask(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.source = source["source"];
	        this.status = source["status"];
	        this.priority = source["priority"];
	        this.tags = source["tags"];
	        this.content_md = source["content_md"];
	        this.result_md = source["result_md"];
	        this.started_at = source["started_at"];
	        this.completed_at = source["completed_at"];
	        this.converted_to_jira = source["converted_to_jira"];
	        this.converted_jira_key = source["converted_jira_key"];
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	    }
	}
	export class Link {
	    title: string;
	    url: string;
	    type: string;
	
	    static createFrom(source: any = {}) {
	        return new Link(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.url = source["url"];
	        this.type = source["type"];
	    }
	}
	export class Issue {
	    id: number;
	    jira_key: string;
	    title: string;
	    status: string;
	    priority: string;
	    tags: string[];
	    summary_md: string;
	    background_md: string;
	    analysis_md: string;
	    solution_md: string;
	    actions_md: string;
	    result_md: string;
	    todo_md: string;
	    links: Link[];
	    started_at: string;
	    completed_at: string;
	    created_at: string;
	    updated_at: string;
	
	    static createFrom(source: any = {}) {
	        return new Issue(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.jira_key = source["jira_key"];
	        this.title = source["title"];
	        this.status = source["status"];
	        this.priority = source["priority"];
	        this.tags = source["tags"];
	        this.summary_md = source["summary_md"];
	        this.background_md = source["background_md"];
	        this.analysis_md = source["analysis_md"];
	        this.solution_md = source["solution_md"];
	        this.actions_md = source["actions_md"];
	        this.result_md = source["result_md"];
	        this.todo_md = source["todo_md"];
	        this.links = this.convertValues(source["links"], Link);
	        this.started_at = source["started_at"];
	        this.completed_at = source["completed_at"];
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Dashboard {
	    recent_issues: Issue[];
	    active_issues: Issue[];
	    temp_tasks: TempTask[];
	    todos: IssueTodo[];
	    week: WeekView;
	
	    static createFrom(source: any = {}) {
	        return new Dashboard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.recent_issues = this.convertValues(source["recent_issues"], Issue);
	        this.active_issues = this.convertValues(source["active_issues"], Issue);
	        this.temp_tasks = this.convertValues(source["temp_tasks"], TempTask);
	        this.todos = this.convertValues(source["todos"], IssueTodo);
	        this.week = this.convertValues(source["week"], WeekView);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	
	
	export class IssueFilter {
	    Query: string;
	    Status: string;
	    Tag: string;
	    Limit: number;
	    Offset: number;
	    All: boolean;
	
	    static createFrom(source: any = {}) {
	        return new IssueFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Query = source["Query"];
	        this.Status = source["Status"];
	        this.Tag = source["Tag"];
	        this.Limit = source["Limit"];
	        this.Offset = source["Offset"];
	        this.All = source["All"];
	    }
	}
	export class IssueSummaryResponse {
	    summary: string;
	    issue: Issue;
	
	    static createFrom(source: any = {}) {
	        return new IssueSummaryResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.summary = source["summary"];
	        this.issue = this.convertValues(source["issue"], Issue);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	
	
	export class SearchResult {
	    type: string;
	    id: string;
	    title: string;
	    snippet: string;
	    url: string;
	    updated_at: string;
	
	    static createFrom(source: any = {}) {
	        return new SearchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.id = source["id"];
	        this.title = source["title"];
	        this.snippet = source["snippet"];
	        this.url = source["url"];
	        this.updated_at = source["updated_at"];
	    }
	}
	
	export class TempTaskEvent {
	    id: number;
	    temp_task_id: number;
	    event_type: string;
	    content_md: string;
	    happened_at: string;
	    created_at: string;
	    updated_at: string;
	
	    static createFrom(source: any = {}) {
	        return new TempTaskEvent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.temp_task_id = source["temp_task_id"];
	        this.event_type = source["event_type"];
	        this.content_md = source["content_md"];
	        this.happened_at = source["happened_at"];
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	    }
	}
	export class TempTaskFilter {
	    Query: string;
	    Status: string;
	    Tag: string;
	    Limit: number;
	    Offset: number;
	    All: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TempTaskFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Query = source["Query"];
	        this.Status = source["Status"];
	        this.Tag = source["Tag"];
	        this.Limit = source["Limit"];
	        this.Offset = source["Offset"];
	        this.All = source["All"];
	    }
	}
	export class TodayWorkflow {
	    date: string;
	    issues: Issue[];
	    temp_tasks: TempTask[];
	    todos: IssueTodo[];
	    done: string[];
	    active: string[];
	    weekly_draft: string;
	    day: DayWork;
	
	    static createFrom(source: any = {}) {
	        return new TodayWorkflow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.issues = this.convertValues(source["issues"], Issue);
	        this.temp_tasks = this.convertValues(source["temp_tasks"], TempTask);
	        this.todos = this.convertValues(source["todos"], IssueTodo);
	        this.done = source["done"];
	        this.active = source["active"];
	        this.weekly_draft = source["weekly_draft"];
	        this.day = this.convertValues(source["day"], DayWork);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class UploadedImage {
	    url: string;
	    filename: string;
	    content_type: string;
	    size: number;
	
	    static createFrom(source: any = {}) {
	        return new UploadedImage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.filename = source["filename"];
	        this.content_type = source["content_type"];
	        this.size = source["size"];
	    }
	}
	

}

