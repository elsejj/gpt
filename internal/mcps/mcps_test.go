package mcps

import (
	"testing"
)

func TestStart(t *testing.T) {

	//fileSystemProvider := "/home/jia/tools/mcp_servers/src/filesystem/index.ts /home/jia/tools/mcp_servers"

	//fileSystemProvider := "/home/jia/temp/py_mcp/server.py"
	fileSystemProvider := "http://127.0.0.1:8000/cnstock/sse"

	s, err := New(fileSystemProvider)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Shutdown()
	t.Log("start")
	for _, tool := range s.Tools {
		t.Log("Tool:", tool)
	}

	// ctx := context.Background()
	// response, err := s.CallTool(ctx, "aaa", "list_directory", map[string]any{
	// 	"path": "/home/jia/tools/mcp_servers",
	// })
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// t.Log("Response:", response)

}
