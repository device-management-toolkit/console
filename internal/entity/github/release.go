/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: Apache-2.0
 **********************************************************************/

package github

// Release represents the structure of the GitHub API response for releases.
type Release struct {
	URL         string  `json:"url"`
	TagName     string  `json:"tag_name"`
	Name        string  `json:"name"`
	Body        string  `json:"body"`
	Prerelease  bool    `json:"prerelease"`
	CreatedAt   string  `json:"created_at"`
	PublishedAt string  `json:"published_at"`
	HTMLURL     string  `json:"html_url"`
	AssetsURL   string  `json:"assets_url"`
	UploadURL   string  `json:"upload_url"`
	Author      Author  `json:"author"`
	Assets      []Asset `json:"assets"`
}

type Author struct {
	Login   string `json:"login"`
	ID      int    `json:"id"`
	URL     string `json:"url"`
	HTMLURL string `json:"html_url"`
}

type Asset struct {
	URL         string `json:"url"`
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Label       string `json:"label"`
	State       string `json:"state"`
	ContentType string `json:"content_type"`
}
