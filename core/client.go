package core

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkboard "github.com/larksuite/oapi-sdk-go/v3/service/board/v1"
	larkdocx "github.com/larksuite/oapi-sdk-go/v3/service/docx/v1"
	larkdrive "github.com/larksuite/oapi-sdk-go/v3/service/drive/v1"
	larkwiki "github.com/larksuite/oapi-sdk-go/v3/service/wiki/v2"
)

type Client struct {
	larkClient    *lark.Client
	authType      string
	tokenProvider UserAccessTokenProvider
}

func NewClient(config FeishuConfig, tokenProvider UserAccessTokenProvider) *Client {
	return &Client{
		larkClient: lark.NewClient(
			config.AppId,
			config.AppSecret,
			lark.WithReqTimeout(60*time.Second),
		),
		authType:      config.AuthType,
		tokenProvider: tokenProvider,
	}
}

func (c *Client) DownloadImage(ctx context.Context, imgToken, outDir string) (string, error) {
	return c.DownloadAsset(ctx, AssetRef{Kind: AssetKindImage, Token: imgToken}, outDir)
}

func (c *Client) DownloadImageRaw(ctx context.Context, imgToken, imgDir string) (string, []byte, error) {
	return c.DownloadAssetRaw(ctx, AssetRef{Kind: AssetKindImage, Token: imgToken}, imgDir)
}

func (c *Client) DownloadAsset(ctx context.Context, asset AssetRef, outDir string) (string, error) {
	filename, data, err := c.DownloadAssetRaw(ctx, asset, outDir)
	if err != nil {
		return asset.Token, err
	}
	if err = os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		return asset.Token, err
	}
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o666)
	if err != nil {
		return asset.Token, err
	}
	defer file.Close()
	_, err = io.Copy(file, bytes.NewReader(data))
	return filename, err
}

func (c *Client) DownloadAssetRaw(
	ctx context.Context,
	asset AssetRef,
	outDir string,
) (string, []byte, error) {
	options, err := c.methodOptions(ctx)
	if err != nil {
		return asset.Token, nil, err
	}
	reader, originalName, err := c.downloadAssetReader(ctx, asset, options)
	if err != nil {
		return asset.Token, nil, err
	}
	buf := new(bytes.Buffer)
	if _, err = io.Copy(buf, reader); err != nil {
		return asset.Token, nil, err
	}
	filename := buildAssetPath(outDir, asset, originalName)
	return filename, buf.Bytes(), nil
}

func (c *Client) GetDocxContent(ctx context.Context, docToken string) (*Document, []*Block, error) {
	options, err := c.methodOptions(ctx)
	if err != nil {
		return nil, nil, err
	}
	resp, err := c.larkClient.Docx.V1.Document.Get(
		ctx,
		larkdocx.NewGetDocumentReqBuilder().DocumentId(docToken).Build(),
		options...,
	)
	if err != nil {
		return nil, nil, err
	}
	blocks, err := c.getDocxBlocks(ctx, docToken, options)
	return convertDocument(resp.Data.Document), blocks, err
}

func (c *Client) GetWikiNodeInfo(ctx context.Context, token string) (*WikiNode, error) {
	options, err := c.methodOptions(ctx)
	if err != nil {
		return nil, err
	}
	req := larkwiki.NewGetNodeSpaceReqBuilder().Token(token).ObjType("wiki").Build()
	resp, err := c.larkClient.Wiki.Space.GetNode(ctx, req, options...)
	if err != nil {
		return nil, err
	}
	return convertWikiNode(resp.Data.Node), nil
}

func (c *Client) GetDriveFolderFileList(
	ctx context.Context,
	pageToken *string,
	folderToken *string,
) ([]*DriveFile, error) {
	options, err := c.methodOptions(ctx)
	if err != nil {
		return nil, err
	}
	files := make([]*DriveFile, 0)
	nextToken := pageToken
	for {
		builder := larkdrive.NewListFileReqBuilder()
		if folderToken != nil {
			builder.FolderToken(*folderToken)
		}
		if nextToken != nil && *nextToken != "" {
			builder.PageToken(*nextToken)
		}
		resp, err := c.larkClient.Drive.V1.File.List(ctx, builder.Build(), options...)
		if err != nil {
			return nil, err
		}
		files = append(files, convertDriveFiles(resp.Data.Files)...)
		if !boolValueOf(resp.Data.HasMore) {
			return files, nil
		}
		nextToken = resp.Data.NextPageToken
	}
}

func (c *Client) GetWikiName(ctx context.Context, spaceID string) (string, error) {
	options, err := c.methodOptions(ctx)
	if err != nil {
		return "", err
	}
	resp, err := c.larkClient.Wiki.Space.Get(
		ctx,
		larkwiki.NewGetSpaceReqBuilder().SpaceId(spaceID).Build(),
		options...,
	)
	if err != nil {
		return "", err
	}
	return valueOf(resp.Data.Space.Name), nil
}

func (c *Client) GetWikiNodeList(ctx context.Context, spaceID string, parentNodeToken *string) ([]*WikiNode, error) {
	options, err := c.methodOptions(ctx)
	if err != nil {
		return nil, err
	}
	nodes := make([]*WikiNode, 0)
	var pageToken *string
	for {
		builder := larkwiki.NewListSpaceNodeReqBuilder().SpaceId(spaceID)
		if parentNodeToken != nil {
			builder.ParentNodeToken(*parentNodeToken)
		}
		if pageToken != nil && *pageToken != "" {
			builder.PageToken(*pageToken)
		}
		resp, err := c.larkClient.Wiki.SpaceNode.List(ctx, builder.Build(), options...)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, convertWikiNodes(resp.Data.Items)...)
		if !boolValueOf(resp.Data.HasMore) {
			return nodes, nil
		}
		pageToken = resp.Data.PageToken
	}
}

func (c *Client) methodOptions(ctx context.Context) ([]larkcore.RequestOptionFunc, error) {
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
	return []larkcore.RequestOptionFunc{larkcore.WithUserAccessToken(token)}, nil
}

func (c *Client) getDocxBlocks(
	ctx context.Context,
	documentID string,
	options []larkcore.RequestOptionFunc,
) ([]*Block, error) {
	blocks := make([]*Block, 0)
	var pageToken string
	for {
		builder := larkdocx.NewListDocumentBlockReqBuilder().DocumentId(documentID)
		if pageToken != "" {
			builder.PageToken(pageToken)
		}
		resp, err := c.larkClient.Docx.V1.DocumentBlock.List(ctx, builder.Build(), options...)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, convertBlocks(resp.Data.Items)...)
		if !boolValueOf(resp.Data.HasMore) {
			return blocks, nil
		}
		pageToken = valueOf(resp.Data.PageToken)
	}
}

func (c *Client) downloadAssetReader(
	ctx context.Context,
	asset AssetRef,
	options []larkcore.RequestOptionFunc,
) (io.Reader, string, error) {
	switch asset.Kind {
	case AssetKindWhiteboard:
		resp, err := c.larkClient.Board.V1.Whiteboard.DownloadAsImage(
			ctx,
			larkboard.NewDownloadAsImageWhiteboardReqBuilder().WhiteboardId(asset.Token).Build(),
			options...,
		)
		if err != nil {
			return nil, "", err
		}
		return resp.File, resp.FileName, nil
	default:
		resp, err := c.larkClient.Drive.V1.Media.Download(
			ctx,
			larkdrive.NewDownloadMediaReqBuilder().FileToken(asset.Token).Build(),
			options...,
		)
		if err != nil {
			return nil, "", err
		}
		return resp.File, resp.FileName, nil
	}
}

func buildAssetPath(outDir string, asset AssetRef, originalName string) string {
	ext := filepath.Ext(originalName)
	if ext == "" && asset.Kind == AssetKindWhiteboard {
		ext = ".png"
	}
	if ext == "" {
		ext = ".bin"
	}
	return fmt.Sprintf("%s/%s%s", outDir, asset.Token, ext)
}
