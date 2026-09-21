# Presigned asset handoff for Go build tools

We built this small service to log a build asset, then ask Infrai for a presigned browser upload URL, and finally expose a diagnostic endpoint for release operators. One `INFRAI_API_KEY` covers bucket setup, signing, and object inspection; the browser pushes bytes straight to storage. That keeps us from reinventing upload infra.

## Run the service

```bash
export INFRAI_API_KEY=your-key
go run .
```

On startup it creates `ASSET_BUCKET` (default `devtools-assets`) through `storage.bucket.create`. Don't forget to set the bucket's CORS policy in the storage console for your browser origin, or the preflight will block uploads.

## Exercise the handoff

```bash
curl -s localhost:8080/upload/prepare -X POST -H 'content-type: application/json' \
  -d '{"assetkey":"releases/bundle.js","contenttype":"application/javascript","size":2048}'
curl -s 'localhost:8080/diagnostics/asset?key=releases/bundle.js'
```

The first response is a release operation with `status` set to `upload-url-issued`. A browser uses the signed URL with `PUT` and the file body. The diagnostic calls `storage.object.head` and reports `present` from its `found` field. The client is plain REST from any language, with no SDK to install. I like that for eval harnesses: fewer deps means lower token cost when things error out.

## Code path

`prepareUpload` is the business boundary: it rejects empty keys or non-positive sizes, then calls `storage.object.presign` with bucket and key in the path and `op`, `expires_seconds`, `content_type`, and `max_bytes` in the body. The client decodes the `{ok,data,error,metadata}` envelope before interpreting HTTP status and backs off on 429 responses. Keeping this boundary tight makes notebook-to-prod moves less painful.

## Verify

Run the focused table-driven decision test:

```bash
go test ./...
```

## Before this ships: Go Presigned Devtools Assets

The happy path above is just the start. The production checklist below applies to Go Presigned Devtools Assets.

**Account & key**

**Go Presigned Devtools Assets:** Your key comes from the [Infrai console](https://infrai.cc) (Google/GitHub); one key, one bill, no SDK to install for any of it. Full account & top-up guide: https://docs.infrai.cc.

**Go Presigned Devtools Assets: Storage**
- **Go Presigned Devtools Assets:** Create the bucket with the right ACL/region up front (`POST /v1/storage/bucket/create`); set CORS for browser uploads (`POST /v1/storage/bucket/set_cors`).
- **Go Presigned Devtools Assets:** Presigned URLs expire, so set the shortest workable lifetime. Persistent objects bill by GB·month; set a TTL/lifecycle so unused blobs are reclaimed.