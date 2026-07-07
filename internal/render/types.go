package render

// Delta describes a coverage change versus a baseline, in percentage points.
type Delta struct {
	// IsNew marks an entity present now but absent from the baseline.
	IsNew bool
	// HasLine is true when a line-% delta could be computed (both sides had
	// valid lines).
	HasLine bool
	LinePP  float64
	// HasBranch is true when a branch-% delta could be computed.
	HasBranch bool
	BranchPP  float64
}

// Folder is one folder group within a workspace.
type Folder struct {
	Path            string
	LinesValid      int
	LinesCovered    int
	BranchesValid   int
	BranchesCovered int
	// Delta is optional; non-nil only when a baseline was supplied.
	Delta *Delta
}

// Workspace is a single reporting unit (e.g. a service or app).
type Workspace struct {
	ID              string
	DisplayName     string
	LinesValid      int
	LinesCovered    int
	BranchesValid   int
	BranchesCovered int
	Tests           int
	HasTests        bool
	Folders         []Folder
	// Delta is optional; non-nil only when a baseline was supplied.
	Delta *Delta
}

// Regression names a workspace whose line coverage decreased versus baseline.
type Regression struct {
	DisplayName string
	OldPercent  float64
	NewPercent  float64
	DropPP      float64
}

// Summary is the complete input to Markdown and HTML.
type Summary struct {
	Workspaces       []Workspace
	ExcludedFiles    int
	IgnoreFileLoaded bool

	// Baseline-related fields; populated only when a baseline was supplied.
	HasBaseline       bool
	TotalDelta        *Delta
	RemovedWorkspaces []string
	Regressions       []Regression
}
