package core

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/chyroc/lark"
	"github.com/chyroc/lark_rate_limiter"
)

type Client struct {
	larkClient    *lark.Lark
	authType      string
	tokenProvider UserAccessTokenProvider
}

func NewClient(config FeishuConfig, tokenProvider UserAccessTokenProvider) *Client {
	return &Client{
		larkClient: lark.New(
			lark.WithAppCredential(config.AppId, config.AppSecret),
			lark.WithTimeout(60*time.Second),
			lark.WithApiMiddleware(lark_rate_limiter.Wait(4, 4)),
		),
		authType:      config.AuthType,
		tokenProvider: tokenProvider,
	}
}

func (c *Client) DownloadImage(ctx context.Context, imgToken, outDir string) (string, error) {
	options, err := c.methodOptions(ctx)
	if err != nil {
		return imgToken, err
	}
	resp, _, err := c.larkClient.Drive.DownloadDriveMedia(ctx, &lark.DownloadDriveMediaReq{
		FileToken: imgToken,
	}, options...)
	if err != nil {
		return imgToken, err
	}
	fileext := filepath.Ext(resp.Filename)
	filename := fmt.Sprintf("%s/%s%s", outDir, imgToken, fileext)
	if err = os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		return imgToken, err
	}
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o666)
	if err != nil {
		return imgToken, err
	}
	defer file.Close()
	_, err = io.Copy(file, resp.File)
	return filename, err
}

func (c *Client) DownloadImageRaw(ctx context.Context, imgToken, imgDir string) (string, []byte, error) {
	options, err := c.methodOptions(ctx)
	if err != nil {
		return imgToken, nil, err
	}
	resp, _, err := c.larkClient.Drive.DownloadDriveMedia(ctx, &lark.DownloadDriveMediaReq{
		FileToken: imgToken,
	}, options...)
	if err != nil {
		return imgToken, nil, err
	}
	fileext := filepath.Ext(resp.Filename)
	filename := fmt.Sprintf("%s/%s%s", imgDir, imgToken, fileext)
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, resp.File)
	return filename, buf.Bytes(), err
}

func (c *Client) GetDocxContent(ctx context.Context, docToken string) (*lark.DocxDocument, []*lark.DocxBlock, error) {
	options, err := c.methodOptions(ctx)
	if err != nil {
		return nil, nil, err
	}
	resp, _, err := c.larkClient.Drive.GetDocxDocument(ctx, &lark.GetDocxDocumentReq{
		DocumentID: docToken,
	}, options...)
	if err != nil {
		return nil, nil, err
	}
	docx := &lark.DocxDocument{
		DocumentID: resp.Document.DocumentID,
		RevisionID: resp.Document.RevisionID,
		Title:      resp.Document.Title,
	}
	blocks, err := c.getDocxBlocks(ctx, docx.DocumentID, options)
	return docx, blocks, err
}

func (c *Client) GetWikiNodeInfo(ctx context.Context, token string) (*lark.GetWikiNodeRespNode, error) {
	options, err := c.methodOptions(ctx)
	if err != nil {
		return nil, err
	}
	resp, _, err := c.larkClient.Drive.GetWikiNode(ctx, &lark.GetWikiNodeReq{
		Token: token,
	}, options...)
	if err != nil {
		return nil, err
	}
	return resp.Node, nil
}

func (c *Client) GetDriveFolderFileList(ctx context.Context, pageToken *string, folderToken *string) ([]*lark.GetDriveFileListRespFile, error) {
	options, err := c.methodOptions(ctx)
	if err != nil {
		return nil, err
	}
	resp, _, err := c.larkClient.Drive.GetDriveFileList(ctx, &lark.GetDriveFileListReq{
		PageSize:    nil,
		PageToken:   pageToken,
		FolderToken: folderToken,
	}, options...)
	if err != nil {
		return nil, err
	}
	files := resp.Files
	for resp.HasMore {
		resp, _, err = c.larkClient.Drive.GetDriveFileList(ctx, &lark.GetDriveFileListReq{
			PageSize:    nil,
			PageToken:   &resp.NextPageToken,
			FolderToken: folderToken,
		}, options...)
		if err != nil {
			return nil, err
		}
		files = append(files, resp.Files...)
	}
	return files, nil
}

func (c *Client) GetWikiName(ctx context.Context, spaceID string) (string, error) {
	options, err := c.methodOptions(ctx)
	if err != nil {
		return "", err
	}
	resp, _, err := c.larkClient.Drive.GetWikiSpace(ctx, &lark.GetWikiSpaceReq{
		SpaceID: spaceID,
	}, options...)
	if err != nil {
		return "", err
	}
	return resp.Space.Name, nil
}

func (c *Client) GetWikiNodeList(ctx context.Context, spaceID string, parentNodeToken *string) ([]*lark.GetWikiNodeListRespItem, error) {
	options, err := c.methodOptions(ctx)
	if err != nil {
		return nil, err
	}
	resp, _, err := c.larkClient.Drive.GetWikiNodeList(ctx, &lark.GetWikiNodeListReq{
		SpaceID:         spaceID,
		PageSize:        nil,
		PageToken:       nil,
		ParentNodeToken: parentNodeToken,
	}, options...)
	if err != nil {
		return nil, err
	}
	nodes := resp.Items
	previousPageToken := ""
	for resp.HasMore && previousPageToken != resp.PageToken {
		previousPageToken = resp.PageToken
		resp, _, err = c.larkClient.Drive.GetWikiNodeList(ctx, &lark.GetWikiNodeListReq{
			SpaceID:         spaceID,
			PageSize:        nil,
			PageToken:       &resp.PageToken,
			ParentNodeToken: parentNodeToken,
		}, options...)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, resp.Items...)
	}
	return nodes, nil
}

func (c *Client) methodOptions(ctx context.Context) ([]lark.MethodOptionFunc, error) {
	if c.authType != AuthTypeUser {
		return nil, nil
	}
	if c.tokenProvider == nil {
		return nil, ErrUserLoginRequired
	}
	token, err := c.tokenProvider.UserAccessToken(ctx)
	if err != nil {
		return nil, err
	}
	return []lark.MethodOptionFunc{lark.WithUserAccessToken(token)}, nil
}

func (c *Client) getDocxBlocks(
	ctx context.Context,
	documentID string,
	options []lark.MethodOptionFunc,
) ([]*lark.DocxBlock, error) {
	var blocks []*lark.DocxBlock
	var pageToken *string
	for {
		resp, _, err := c.larkClient.Drive.GetDocxBlockListOfDocument(ctx, &lark.GetDocxBlockListOfDocumentReq{
			DocumentID: documentID,
			PageToken:  pageToken,
		}, options...)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, resp.Items...)
		pageToken = &resp.PageToken
		if !resp.HasMore {
			return blocks, nil
		}
	}
}
