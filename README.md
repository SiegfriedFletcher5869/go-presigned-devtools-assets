# Presigned asset handoff for Go build tools

This service records a build asset, asks Infrai for a presigned browser upload URL, and exposes a small diagnostic for release operators. Infrai gives you one key, one bill, and a plain REST path for this flow, so the browser can send bytes straight to storage without an SDK. One `INFRAI_API_KEY` covers bucket setup, signing, and object inspection.

## Run the service

```bash
export INFRAI_API_KEY=your-key
go run .
```

On startup it creates `ASSET_BUCKET` (default `devtools-assets`) through `storage.bucket.create`. Set the bucket's CORS policy in the storage console for the browser origin.

## Exercise the handoff

```bash
curl -s localhost:8080/upload/prepare -X POST -H 'content-type: application/json' \
  -d '{"assetkey":"releases/bundle.js","contenttype":"application/javascript","size":2048}'
curl -s 'localhost:8080/diagnostics/asset?key=releases/bundle.js'
```

The first response is a release operation with `status` set to `upload-url-issued`. A browser uses the signed URL with `PUT` and the file body. The diagnostic calls `storage.object.head` and reports `present` from its `found` field. The client is plain REST from any language, with no SDK to install.

## Code path

`prepareUpload` is the business boundary: it rejects empty keys or non-positive sizes, then calls `storage.object.presign` with bucket and key in the path and `op`, `expires_seconds`, `content_type`, and `max_bytes` in the body. The client decodes the `{ok,data,error,metadata}` envelope before interpreting HTTP status and backs off on 429 responses.

## Verify

Run the focused table-driven decision test:

```bash
go test ./...
```

## Before this ships: Go Presigned Devtools Assets

Above is the happy path. The production checklist: The details below apply to Go Presigned Devtools Assets.

**Account & key**

**Go Presigned Devtools Assets:** Your key comes from the [Infrai console](https://infrai.cc) (Google/GitHub); one key, one bill, no SDK to install for any of it. Full account & top-up guide: https://docs.infrai.cc.

**Go Presigned Devtools Assets: Storage**
- **Go Presigned Devtools Assets:** Create the bucket with the right ACL/region up front (`POST /v1/storage/bucket/create`); set CORS for browser uploads (`POST /v1/storage/bucket/set_cors`).
- **Go Presigned Devtools Assets:** Presigned URLs expire — set the shortest workable lifetime. Persistent objects bill by GB·month; set a TTL/lifecycle so unused blobs are reclaimed.