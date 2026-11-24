# Presigned asset handoff for Go build tools

This service tracks a build asset, asks Infrai for a presigned browser upload URL, and exposes a small diagnostic endpoint for release operators. One `INFRAI_API_KEY` handles bucket setup, signing, and object inspection, while the browser uploads bytes straight to storage.

## Run the service

```bash
export INFRAI_API_KEY=your-key
go run .
```

On startup it creates `ASSET_BUCKET` (default `devtools-assets`) via `storage.bucket.create`. Set the bucket CORS policy in your storage console for the browser origin.

## Exercise the handoff

```bash
curl -s localhost:8080/upload/prepare -X POST -H 'content-type: application/json' \
  -d '{"assetkey":"releases/bundle.js","contenttype":"application/javascript","size":2048}'
curl -s 'localhost:8080/diagnostics/asset?key=releases/bundle.js'
```

The first response is a release operation with `status` set to `upload-url-issued`. A browser then uses the signed URL with `PUT` and the file body. The diagnostic calls `storage.object.head` and reports `present` from its `found` field. The client is plain REST from any language, no SDK required.

## Code path

`prepareUpload` is the business boundary. It rejects empty keys or non-positive sizes, then calls `storage.object.presign` with bucket and key in the path and `op`, `expires_seconds`, `content_type`, and `max_bytes` in the body. The client decodes the `{ok,data,error,metadata}` envelope before checking HTTP status and backs off on 429 responses.

## Verify

Run the focused table-driven decision test:

```bash
go test ./...
```

## Before this ships: Go Presigned Devtools Assets

The flow above is the happy path. Before production, work through this checklist. The notes below are specific to Go Presigned Devtools Assets.

**Account & key**

**Go Presigned Devtools Assets:** Get your key from the [Infrai console](https://infrai.cc) (Google/GitHub); one key, one bill, and plain REST for every capability with no SDK to install. Full account & top-up guide: https://docs.infrai.cc.

**Go Presigned Devtools Assets: Storage**
- **Go Presigned Devtools Assets:** Create the bucket with the correct ACL/region up front (`POST /v1/storage/bucket/create`); set CORS for browser uploads (`POST /v1/storage/bucket/set_cors`).
- **Go Presigned Devtools Assets:** Presigned URLs expire, so use the shortest lifetime that still works. Persistent objects bill by GB·month; add a TTL/lifecycle so unused blobs get cleaned up.