package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/Wsine/feishu2md/core"
	"github.com/Wsine/feishu2md/utils"
	"github.com/gin-gonic/gin"
)

func downloadHandler(c *gin.Context) {
	if requireWebLogin(c) {
		return
	}

	// get parameters
	feishu_docx_url, err := url.QueryUnescape(c.Query("url"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid encoded feishu/larksuite URL")
		return
	}

	// Validate the url to download
	docType, docToken, err := utils.ValidateDocumentURL(feishu_docx_url)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	fmt.Println("Captured document token:", docToken)

	// Create client with context
	ctx := c.Request.Context()
	client, err := newWebClient(c)
	if err != nil {
		c.String(http.StatusUnauthorized, err.Error())
		return
	}
	outputConfig := core.NewConfig("", "").Output

	// Process the download
	parser := core.NewParser(outputConfig)
	markdown := ""

	// for a wiki page, we need to renew docType and docToken first
	if docType == "wiki" {
		node, err := client.GetWikiNodeInfo(ctx, docToken)
		if err != nil {
			failDownload(c, http.StatusBadGateway, "Internal error: client.GetWikiNodeInfo", err)
			return
		}
		docType = node.ObjType
		docToken = node.ObjToken
	}
	if docType == "docs" {
		c.String(http.StatusBadRequest, "Unsupported docs document type")
		return
	}

	docx, blocks, err := client.GetDocxContent(ctx, docToken)
	if err != nil {
		failDownload(c, http.StatusBadGateway, "Internal error: client.GetDocxContent", err)
		return
	}
	markdown = parser.ParseDocxContent(docx, blocks)

	zipBuffer := new(bytes.Buffer)
	writer := zip.NewWriter(zipBuffer)
	for _, asset := range parser.Assets {
		localLink, rawImage, err := client.DownloadAssetRaw(ctx, asset, outputConfig.ImageDir)
		if err != nil {
			failDownload(c, http.StatusBadGateway, "Internal error: client.DownloadImageRaw", err)
			return
		}
		markdown = strings.Replace(markdown, asset.Token, localLink, 1)
		f, err := writer.Create(localLink)
		if err != nil {
			failDownload(c, http.StatusInternalServerError, "Internal error: zipWriter.Create", err)
			return
		}
		_, err = f.Write(rawImage)
		if err != nil {
			failDownload(c, http.StatusInternalServerError, "Internal error: zipWriter.Create.Write", err)
			return
		}
	}

	result := markdown

	// Set response
	if len(parser.Assets) > 0 {
		mdName := fmt.Sprintf("%s.md", docToken)
		f, err := writer.Create(mdName)
		if err != nil {
			failDownload(c, http.StatusInternalServerError, "Internal error: zipWriter.Create", err)
			return
		}
		_, err = f.Write([]byte(result))
		if err != nil {
			failDownload(c, http.StatusInternalServerError, "Internal error: zipWriter.Create.Write", err)
			return
		}

		err = writer.Close()
		if err != nil {
			failDownload(c, http.StatusInternalServerError, "Internal error: zipWriter.Close", err)
			return
		}
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.zip"`, docToken))
		c.Data(http.StatusOK, "application/octet-stream", zipBuffer.Bytes())
	} else {
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.md"`, docToken))
		c.Data(http.StatusOK, "application/octet-stream", []byte(result))
	}
}

func failDownload(c *gin.Context, status int, message string, err error) {
	_ = c.Error(err)
	c.String(status, message)
}
