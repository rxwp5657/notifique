package dto

type DistributionList struct {
	Name       string   `json:"name" binding:"max=120,min=3"`
	Recipients []string `json:"recipients" binding:"max=256,unique,dive,min=1"`
}

type DistributionListSummary struct {
	Name               string `json:"name"`
	NumberOfRecipients int    `json:"numberOfRecipients"`
}

type DistributionListRecipients struct {
	Recipients []string `json:"recipients" binding:"unique,max=256,dive,min=1"`
}

type DistributionListUriParams struct {
	Name string `uri:"id" binding:"max=120,min=3"`
}
