package catalog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexandrmotologa/toolcraft/internal/domain"
)

// ExportCatalog generates a human and agent readable documentation catalog in Markdown or HTML.
func ExportCatalog(spec *domain.Spec, format, outputPath string) error {
	var content string
	switch strings.ToLower(format) {
	case "html":
		content = GenerateHTMLCatalog(spec)
	default:
		content = GenerateMarkdownCatalog(spec)
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil && filepath.Dir(outputPath) != "." {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return os.WriteFile(outputPath, []byte(content), 0644)
}

// GenerateMarkdownCatalog builds a comprehensive GitHub Flavored Markdown catalog.
func GenerateMarkdownCatalog(spec *domain.Spec) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("# %s — MCP Tools Catalog\n\n", spec.Title))
	b.WriteString(fmt.Sprintf("> Version: %s | Base URL: `%s`\n\n", spec.Version, spec.BaseURL))
	if spec.Description != "" {
		b.WriteString(fmt.Sprintf("%s\n\n", spec.Description))
	}

	b.WriteString("## Available Tools Index\n\n")
	b.WriteString("| Tool Name | HTTP Method | Endpoint Path | Safety Tier | Required Inputs |\n")
	b.WriteString("|---|---|---|---|---|\n")

	for _, tool := range spec.Tools {
		safety := "Mutating"
		if tool.ReadOnly {
			safety = "Read-Only"
		}

		var reqInputs []string
		for _, p := range tool.Parameters {
			if p.Required {
				reqInputs = append(reqInputs, p.Name)
			}
		}
		if tool.RequestBody != nil {
			for _, p := range tool.RequestBody.Properties {
				if p.Required {
					reqInputs = append(reqInputs, p.Name)
				}
			}
		}

		reqStr := "-"
		if len(reqInputs) > 0 {
			reqStr = strings.Join(reqInputs, ", ")
		}

		b.WriteString(fmt.Sprintf("| [`%s`](#%s) | `%s` | `%s` | %s | %s |\n",
			tool.ID, strings.ToLower(tool.ID), tool.Method, tool.Path, safety, reqStr))
	}

	b.WriteString("\n---\n\n## Tool Specifications\n\n")

	for _, tool := range spec.Tools {
		b.WriteString(fmt.Sprintf("### `%s`\n\n", tool.ID))
		b.WriteString(fmt.Sprintf("- **Endpoint**: `%s %s`\n", tool.Method, tool.Path))
		b.WriteString(fmt.Sprintf("- **Read-Only**: `%v`\n", tool.ReadOnly))
		b.WriteString(fmt.Sprintf("- **Agent Description**: %s\n\n", tool.Description))

		allParams := make([]domain.ParameterDefinition, 0, len(tool.Parameters))
		allParams = append(allParams, tool.Parameters...)
		if tool.RequestBody != nil {
			allParams = append(allParams, tool.RequestBody.Properties...)
		}

		if len(allParams) > 0 {
			b.WriteString("#### Input Schema Parameters\n\n")
			b.WriteString("| Parameter | Location | Type | Required | Description |\n")
			b.WriteString("|---|---|---|---|---|\n")

			sampleArgs := make(map[string]interface{})
			for _, p := range allParams {
				reqStr := "No"
				if p.Required {
					reqStr = "**Yes**"
				}
				desc := p.Description
				if desc == "" {
					desc = "-"
				}
				b.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` | %s | %s |\n", p.Name, p.In, p.Type, reqStr, desc))

				if p.Required {
					sampleArgs[p.Name] = sampleValue(p.Type, p.Name)
				}
			}

			b.WriteString("\n#### Sample MCP Tool Call\n\n")
			sampleCall := map[string]interface{}{
				"name":      tool.ID,
				"arguments": sampleArgs,
			}
			callBytes, _ := json.MarshalIndent(sampleCall, "", "  ")
			b.WriteString("```json\n" + string(callBytes) + "\n```\n\n")
		} else {
			b.WriteString("*This tool takes no parameters.*\n\n")
		}

		b.WriteString("---\n\n")
	}

	return b.String()
}

// GenerateHTMLCatalog builds a self-contained responsive HTML page with instant live search.
func GenerateHTMLCatalog(spec *domain.Spec) string {
	toolsJSON, _ := json.Marshal(spec.Tools)

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>%s - MCP Tools Catalog</title>
  <style>
    :root {
      --bg: #0f111a;
      --card-bg: #1a1c29;
      --text: #e2e8f0;
      --muted: #94a3b8;
      --primary: #6366f1;
      --border: #2d3748;
      --green: #10b981;
      --blue: #3b82f6;
      --amber: #f59e0b;
      --red: #ef4444;
    }
    * { box-sizing: border-box; margin: 0; padding: 0; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, monospace; }
    body { background: var(--bg); color: var(--text); padding: 2rem; max-width: 1200px; margin: 0 auto; line-height: 1.6; }
    header { margin-bottom: 2rem; border-bottom: 1px solid var(--border); padding-bottom: 1rem; }
    h1 { font-size: 2rem; margin-bottom: 0.5rem; color: #fff; }
    .meta { color: var(--muted); font-size: 0.9rem; }
    .search-box { width: 100%%; padding: 0.8rem 1rem; background: var(--card-bg); border: 1px solid var(--border); border-radius: 8px; color: #fff; font-size: 1rem; margin-bottom: 2rem; outline: none; }
    .search-box:focus { border-color: var(--primary); }
    .grid { display: grid; grid-template-columns: 1fr; gap: 1.5rem; }
    .card { background: var(--card-bg); border: 1px solid var(--border); border-radius: 8px; padding: 1.5rem; }
    .badge { display: inline-block; padding: 0.2rem 0.5rem; border-radius: 4px; font-weight: bold; font-size: 0.8rem; margin-right: 0.5rem; }
    .badge-get { background: var(--blue); color: #000; }
    .badge-post { background: var(--green); color: #000; }
    .badge-delete { background: var(--red); color: #fff; }
    .badge-put, .badge-patch { background: var(--amber); color: #000; }
    .badge-readonly { border: 1px solid var(--green); color: var(--green); }
    .badge-mutating { border: 1px solid var(--amber); color: var(--amber); }
    .tool-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 1rem; }
    .tool-name { font-size: 1.25rem; font-weight: bold; color: #fff; }
    .desc { color: var(--text); margin-bottom: 1rem; font-size: 0.95rem; }
    table { width: 100%%; border-collapse: collapse; margin-top: 1rem; font-size: 0.9rem; }
    th, td { text-align: left; padding: 0.5rem; border-bottom: 1px solid var(--border); }
    th { color: var(--muted); }
    code { background: #111; padding: 0.2rem 0.4rem; border-radius: 4px; font-size: 0.85rem; }
  </style>
</head>
<body>
  <header>
    <h1>%s — MCP Tools Catalog</h1>
    <div class="meta">Version: %s | Base URL: %s | Total Tools: %d</div>
  </header>

  <input type="text" id="searchInput" class="search-box" placeholder="Search tools by name, endpoint path, or method (e.g. 'get', 'pets', 'invoice')...">

  <div id="toolsContainer" class="grid"></div>

  <script>
    const tools = %s;
    const container = document.getElementById('toolsContainer');
    const searchInput = document.getElementById('searchInput');

    function renderTools(list) {
      if (list.length === 0) {
        container.innerHTML = '<div style="color: var(--muted);">No matching tools found.</div>';
        return;
      }
      container.innerHTML = list.map(t => {
        const methodClass = 'badge-' + t.method.toLowerCase();
        const safetyBadge = t.readOnly ? '<span class="badge badge-readonly">Read-Only</span>' : '<span class="badge badge-mutating">Mutating</span>';
        
        let paramsHtml = '';
        const allParams = [...(t.parameters || []), ...((t.requestBody && t.requestBody.properties) || [])];
        if (allParams.length > 0) {
          paramsHtml = '<table><thead><tr><th>Name</th><th>In</th><th>Type</th><th>Required</th><th>Description</th></tr></thead><tbody>' +
            allParams.map(p => '<tr><td><code>' + p.name + '</code></td><td>' + p.in + '</td><td>' + p.type + '</td><td>' + (p.required ? '<strong>Yes</strong>' : 'No') + '</td><td>' + (p.description || '-') + '</td></tr>').join('') +
            '</tbody></table>';
        } else {
          paramsHtml = '<p style="color: var(--muted); font-size: 0.85rem; margin-top: 0.5rem;">No input parameters required.</p>';
        }

        return '<div class="card">' +
          '<div class="tool-header">' +
            '<div><span class="badge ' + methodClass + '">' + t.method + '</span> <span class="tool-name">' + t.id + '</span></div>' +
            '<div>' + safetyBadge + '</div>' +
          '</div>' +
          '<div style="color: var(--muted); font-size: 0.85rem; margin-bottom: 0.5rem;"><code>' + t.path + '</code></div>' +
          '<div class="desc">' + t.description + '</div>' +
          paramsHtml +
        '</div>';
      }).join('');
    }

    renderTools(tools);

    searchInput.addEventListener('input', (e) => {
      const q = e.target.value.toLowerCase().trim();
      const filtered = tools.filter(t => 
        t.id.toLowerCase().includes(q) ||
        t.path.toLowerCase().includes(q) ||
        t.method.toLowerCase().includes(q) ||
        t.description.toLowerCase().includes(q)
      );
      renderTools(filtered);
    });
  </script>
</body>
</html>`, spec.Title, spec.Title, spec.Version, spec.BaseURL, len(spec.Tools), string(toolsJSON))
}

func sampleValue(pType, name string) interface{} {
	switch pType {
	case "integer":
		return 10
	case "number":
		return 19.99
	case "boolean":
		return true
	default:
		return "sample_" + name
	}
}
