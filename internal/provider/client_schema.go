package provider

type Project struct {
	Id           int64  `json:"id"`
	Name         string `json:"name"`
	Organization string `json:"organization"`
	ApiToken     string `json:"api_token"`
}

type ProjectCreateInput struct {
	Name string `json:"name"`
}

type Experiment struct {
	Id          int64                `json:"id"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	FeatureFlag FeatureFlag          `json:"feature_flag"`
	Parameters  ExperimentParameters `json:"parameters"`
	StartDate   *string              `json:"start_date"`
	EndDate     *string              `json:"end_date"`
	Deleted     bool                 `json:"deleted"`
}

type FeatureFlag struct {
	Id  int64  `json:"id"`
	Key string `json:"key"`
}

type ExperimentParameters struct {
	FeatureFlagVariants []FeatureFlagVariant `json:"feature_flag_variants"`
}

type FeatureFlagVariant struct {
	Key               string `json:"key"`
	RolloutPercentage int    `json:"rollout_percentage"`
}

type ExperimentCreateInput struct {
	Name           string               `json:"name"`
	Description    string               `json:"description,omitempty"`
	FeatureFlagKey string               `json:"feature_flag_key"`
	Parameters     ExperimentParameters `json:"parameters"`
	StartDate      *string              `json:"start_date,omitempty"`
	EndDate        *string              `json:"end_date,omitempty"`
}

type ExperimentUpdateInput struct {
	Name        string               `json:"name"`
	Description string               `json:"description,omitempty"`
	Parameters  ExperimentParameters `json:"parameters"`
	StartDate   *string              `json:"start_date,omitempty"`
	EndDate     *string              `json:"end_date,omitempty"`
}

type ExperimentDeleteInput struct {
	Deleted bool `json:"deleted"`
}
