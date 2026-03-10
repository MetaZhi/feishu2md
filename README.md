# feishu2md

[![Golang - feishu2md](https://img.shields.io/github/go-mod/go-version/wsine/feishu2md?color=%2376e1fe&logo=go)](https://go.dev/)
[![Unittest](https://github.com/Wsine/feishu2md/actions/workflows/unittest.yaml/badge.svg)](https://github.com/Wsine/feishu2md/actions/workflows/unittest.yaml)
[![Release](https://img.shields.io/github/v/release/wsine/feishu2md?color=orange&logo=github)](https://github.com/Wsine/feishu2md/releases)
[![Docker - feishu2md](https://img.shields.io/badge/Docker-feishu2md-2496ed?logo=docker&logoColor=white)](https://hub.docker.com/r/wwwsine/feishu2md)
[![Render - feishu2md](https://img.shields.io/badge/Render-feishu2md-4cfac9?logo=render&logoColor=white)](https://feishu2md.onrender.com)
![Last Review](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fbadge-last-review.wsine.workers.dev%2FWsine%2Ffeishu2md&query=%24.reviewed_at&label=last%20review)

这是一个下载飞书文档为 Markdown 文件的工具，使用 Go 语言实现。

**请看这里：由于原作者已不再使用飞书文档，项目转为社区维护，欢迎 PR，有能力的维护者会被选择为主协调员。**

## 动机

[《一日一技 | 我开发的这款小工具，轻松助你将飞书文档转为 Markdown》](https://sspai.com/post/73386)

## 获取 API Token

配置文件需要填写 APP ID 和 APP SECRET 信息，请参考 [飞书官方文档](https://open.feishu.cn/document/ukTMukTMukTM/ukDNz4SO0MjL5QzM/get-) 获取。推荐设置为

- 进入飞书[开发者后台](https://open.feishu.cn/app)
- 创建企业自建应用（个人版），信息随意填写
- （重要）打开权限管理，开通以下必要的权限（可点击以下链接参考 API 调试台->权限配置字段）
  - [获取文档基本信息](https://open.feishu.cn/document/server-docs/docs/docs/docx-v1/document/get)，「查看新版文档」权限 `docx:document:readonly`
  - [获取文档所有块](https://open.feishu.cn/document/server-docs/docs/docs/docx-v1/document/list)，「查看新版文档」权限 `docx:document:readonly`
  - [下载素材](https://open.feishu.cn/document/server-docs/docs/drive-v1/media/download)，「下载云文档中的图片和附件」权限 `docs:document.media:download`
  - [获取文件夹中的文件清单](https://open.feishu.cn/document/server-docs/docs/drive-v1/folder/list)，「查看、评论、编辑和管理云空间中所有文件」权限 `drive:file:readonly`
  - [获取知识空间节点信息](https://open.feishu.cn/document/server-docs/docs/wiki-v2/space-node/get_node)，「查看知识库」权限 `wiki:wiki:readonly`
- 打开凭证与基础信息，获取 App ID 和 App Secret

## 用户登录授权

从当前版本开始，`feishu2md` 支持内建的飞书用户登录授权，不再需要手动在外部获取 `user_access_token`。

### CLI 用法

1. 先配置应用凭证，并切换到用户鉴权模式：

   ```bash
   feishu2md config \
     --appId "<your_app_id>" \
     --appSecret "<your_app_secret>" \
     --authType "user" \
     --redirectURI "http://127.0.0.1:38080/auth/callback"
   ```

2. 在飞书开放平台里，将上面的 `redirectURI` 配置到应用的重定向地址。

3. 执行登录：

   ```bash
   feishu2md login
   ```

4. 按终端输出的链接完成飞书登录。登录成功后，工具会自动保存 `access_token` / `refresh_token`，后续下载时自动刷新，不需要每 2 小时手动更新。

### Web 用法

Web 服务默认仍兼容应用鉴权。若要启用用户登录授权，请设置：

```bash
FEISHU_AUTH_TYPE=user
FEISHU_APP_ID=<your_app_id>
FEISHU_APP_SECRET=<your_app_secret>
FEISHU_OAUTH_REDIRECT_URI=http://127.0.0.1:8080/auth/callback
```

然后在飞书开放平台为该应用配置同样的重定向地址。访问 Web 页面后，点击 `Login With Feishu`，登录后的下载会基于当前用户的文档/知识库权限执行。

## 如何使用

注意：飞书旧版文档的下载工具已决定不再维护，但分支 [v1_support](https://github.com/Wsine/feishu2md/tree/v1_support) 仍可使用，对应的归档为 [v1.4.0](https://github.com/Wsine/feishu2md/releases/tag/v1.4.0)，请知悉。

<details>
  <summary>命令行版本</summary>

  借助 Go 语言跨平台的特性，已编译好了主要平台的可执行文件，可以在 [Release](https://github.com/Wsine/feishu2md/releases) 中下载，并将相应平台的 feishu2md 可执行文件放置在 PATH 路径中即可。

   **查阅帮助文档**

   ```bash
   $ feishu2md -h
   NAME:
     feishu2md - Download feishu/larksuite document to markdown file

   USAGE:
     feishu2md [global options] command [command options] [arguments...]

   VERSION:
     v2-0e25fa5

   COMMANDS:
     config        Read config file or set field(s) if provided
     login         Login with your Feishu account and persist refreshable user tokens
     download, dl  Download feishu/larksuite document to markdown file
     help, h       Shows a list of commands or help for one command

   GLOBAL OPTIONS:
     --help, -h     show help (default: false)
     --version, -v  print the version (default: false)

   $ feishu2md config -h
   NAME:
      feishu2md config - Read config file or set field(s) if provided

   USAGE:
      feishu2md config [command options] [arguments...]

   OPTIONS:
      --appId value        Set app id for the OPEN API
      --appSecret value    Set app secret for the OPEN API
      --authType value     Set authentication type: 'app' or 'user'
      --redirectURI value  Set oauth redirect uri for built-in user login
      --help, -h           show help (default: false)

   $ feishu2md login -h
   NAME:
      feishu2md login - Login with your Feishu account and persist refreshable user tokens

   USAGE:
      feishu2md login [command options] [arguments...]

   OPTIONS:
      --timeout value  Wait timeout for oauth callback (default: 5m0s)
      --help, -h       show help (default: false)

   $ feishu2md dl -h
   NAME:
     feishu2md download - Download feishu/larksuite document to markdown file
 
   USAGE:
     feishu2md download [command options] <url>
 
   OPTIONS:
     --output value, -o value  Specify the output directory for the markdown files (default: "./")
     --dump                    Dump json response of the OPEN API (default: false)
     --batch                   Download all documents under a folder (default: false)
     --wiki                    Download all documents within the wiki. (default: false)
     --help, -h                show help (default: false)

   ```

   **生成配置文件**

   通过 `feishu2md config --appId <your_id> --appSecret <your_secret>` 命令即可生成该工具的配置文件。

   若希望按登录用户的权限下载私有文档/知识库，请额外执行 `feishu2md login` 完成一次飞书授权。

   通过 `feishu2md config` 命令可以查看配置文件路径以及是否成功配置。

   更多的配置选项请手动打开配置文件更改。

   **下载单个文档为 Markdown**

   通过 `feishu2md dl <your feishu docx url>` 直接下载，文档链接可以通过 **分享 > 开启链接分享 > 互联网上获得链接的人可阅读 > 复制链接** 获得。

   示例：

   ```bash
   $ feishu2md dl "https://domain.feishu.cn/docx/docxtoken"
   ```

  **批量下载某文件夹内的全部文档为 Markdown**

  此功能暂时不支持Docker版本

  通过`feishu2md dl --batch <your feishu folder url>` 直接下载，文件夹链接可以通过 **分享 > 开启链接分享 > 互联网上获得链接的人可阅读 > 复制链接** 获得。

  示例：

  ```bash
  $ feishu2md dl --batch -o output_directory "https://domain.feishu.cn/drive/folder/foldertoken"
  ```

  **批量下载某知识库的全部文档为 Markdown**

  通过`feishu2md dl --wiki <your feishu wiki setting url>` 直接下载，wiki settings链接可以通过 打开知识库设置获得。

  示例：

  ```bash
  $ feishu2md dl --wiki -o output_directory "https://domain.feishu.cn/wiki/settings/123456789101112"
  ```

</details>

<details>
  <summary>Docker版本</summary>

  Docker 镜像：https://hub.docker.com/r/wwwsine/feishu2md

   Docker 命令：`docker run -it --rm -p 8080:8080 -e FEISHU_APP_ID=<your id> -e FEISHU_APP_SECRET=<your secret> -e FEISHU_AUTH_TYPE=user -e FEISHU_OAUTH_REDIRECT_URI=http://127.0.0.1:8080/auth/callback -e GIN_MODE=release wwwsine/feishu2md`

   Docker Compose:

   ```yml
   # docker-compose.yml
   version: '3'
   services:
     feishu2md:
       image: wwwsine/feishu2md
       environment:
         FEISHU_APP_ID: <your id>
         FEISHU_APP_SECRET: <your secret>
         FEISHU_AUTH_TYPE: user
         FEISHU_OAUTH_REDIRECT_URI: http://127.0.0.1:8080/auth/callback
         GIN_MODE: release
       ports:
         - "8080:8080"
   ```

   启动服务 `docker compose up -d`

   然后访问 https://127.0.0.1:8080 粘贴文档链接即可，文档链接可以通过 **分享 > 开启链接分享 > 复制链接** 获得。
</details>

## 感谢

- [chyroc/lark](https://github.com/chyroc/lark)
- [chyroc/lark_docs_md](https://github.com/chyroc/lark_docs_md)
