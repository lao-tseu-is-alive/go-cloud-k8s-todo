#!/bin/bash
buf dep update api/proto
buf generate api/proto
rsync -av api/openapitemplate_4_your_project_name.yaml doc/openapi.json
rsync -av api/openapitemplate_4_your_project_name.yaml cmd/template4YourProjectNameServer/template4YourProjectNameFront/dist/oapidoc/openapi.json