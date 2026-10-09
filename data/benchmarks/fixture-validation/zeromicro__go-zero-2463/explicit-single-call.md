# zeromicro__go-zero-2463 explicit semedit recipe

At the go-zero repository root for base commit `63cfe60f1a7583865757780ebfe2a5309a542134`, add global log fields to `logx` and expose context-aware wrappers in `logc`.

Call `semantic_batch` once with this exact JSON argument. Keep the edit order unchanged: the first child scaffolds `core/logc/logs.go`, and subsequent children build on earlier declarations. The final import operation explicitly adds the `logx` package to the new wrapper file.

```json
{
  "auto_organize_imports": true,
  "trust_workspace": true,
  "edits": [
    {
      "tool": "semantic_scaffold_file",
      "params": {
        "file": "core/logc/logs.go",
        "package": "logc",
        "purpose_header": "// Package logc provides context-aware logging wrappers."
      }
    },
    {
      "tool": "semantic_insert_decl",
      "params": {
        "file": "core/logx/fields.go",
        "source": "var globalFields atomic.Value",
        "auto_organize_imports": true
      }
    },
    {
      "tool": "semantic_insert_decl",
      "params": {
        "file": "core/logx/fields.go",
        "source": "var globalFieldsLock sync.Mutex",
        "auto_organize_imports": true
      }
    },
    {
      "tool": "semantic_insert_function",
      "params": {
        "file": "core/logx/fields.go",
        "source": "func AddGlobalFields(fields ...LogField) {\n\tglobalFieldsLock.Lock()\n\tdefer globalFieldsLock.Unlock()\n\n\told := globalFields.Load()\n\tif old == nil {\n\t\tglobalFields.Store(append([]LogField(nil), fields...))\n\t} else {\n\t\tglobalFields.Store(append(old.([]LogField), fields...))\n\t}\n}",
        "auto_organize_imports": true
      }
    },
    {
      "tool": "semantic_insert_function",
      "params": {
        "file": "core/logx/writer.go",
        "source": "func combineGlobalFields(fields []LogField) []LogField {\n\tglobals := globalFields.Load()\n\tif globals == nil {\n\t\treturn fields\n\t}\n\n\treturn append(globals.([]LogField), fields...)\n}",
        "auto_organize_imports": true
      }
    },
    {
      "tool": "semantic_replace_body",
      "params": {
        "file": "core/logx/writer.go",
        "symbol": "output",
        "body": "fields = combineGlobalFields(fields)\n\nswitch atomic.LoadUint32(&encoding) {\ncase plainEncodingType:\n\twritePlainAny(writer, level, val, buildFields(fields...)...)\ndefault:\n\tentry := make(logEntry)\n\tfor _, field := range fields {\n\t\tentry[field.Key] = field.Value\n\t}\n\tentry[timestampKey] = getTimestamp()\n\tentry[levelKey] = level\n\tentry[contentKey] = val\n\twriteJson(writer, entry)\n}"
      }
    },
    {
      "tool": "semantic_insert_construct",
      "params": {
        "file": "core/logc/logs.go",
        "kind": "type",
        "access_modifier": "public",
        "source": "type (\n\tLogConf = logx.LogConf\n\tLogField = logx.LogField\n)",
        "auto_organize_imports": true
      }
    },
    {
      "tool": "semantic_insert_function",
      "params": {
        "file": "core/logc/logs.go",
        "source": "// AddGlobalFields adds global fields.\nfunc AddGlobalFields(fields ...LogField) {\n\tlogx.AddGlobalFields(fields...)\n}",
        "auto_organize_imports": true
      }
    },
    {
      "tool": "semantic_insert_function",
      "params": {
        "file": "core/logc/logs.go",
        "source": "// Alert alerts v in alert level, and the message is written to error log.\nfunc Alert(_ context.Context, v string) {\n\tlogx.Alert(v)\n}",
        "auto_organize_imports": true
      }
    },
    {
      "tool": "semantic_insert_function",
      "params": {
        "file": "core/logc/logs.go",
        "source": "// Close closes the logging.\nfunc Close() error {\n\treturn logx.Close()\n}",
        "auto_organize_imports": true
      }
    },
    {
      "tool": "semantic_insert_function",
      "params": {
        "file": "core/logc/logs.go",
        "source": "// Error writes v into error log.\nfunc Error(ctx context.Context, v ...interface{}) {\n\tgetLogger(ctx).Error(v...)\n}",
        "auto_organize_imports": true
      }
    },
    {
      "tool": "semantic_insert_function",
      "params": {
        "file": "core/logc/logs.go",
        "source": "// Errorf writes v with format into error log.\nfunc Errorf(ctx context.Context, format string, v ...interface{}) {\n\tgetLogger(ctx).Errorf(fmt.Errorf(format, v...).Error())\n}",
        "auto_organize_imports": true
      }
    },
    {
      "tool": "semantic_insert_function",
      "params": {
        "file": "core/logc/logs.go",
        "source": "// Errorv writes v into error log with json content.\n// No call stack attached, because not elegant to pack the messages.\nfunc Errorv(ctx context.Context, v interface{}) {\n\tgetLogger(ctx).Errorv(v)\n}",
        "auto_organize_imports": true
      }
    },
    {
      "tool": "semantic_insert_function",
      "params": {
        "file": "core/logc/logs.go",
        "source": "// Errorw writes msg along with fields into error log.\nfunc Errorw(ctx context.Context, msg string, fields ...LogField) {\n\tgetLogger(ctx).Errorw(msg, fields...)\n}",
        "auto_organize_imports": true
      }
    },
    {
      "tool": "semantic_insert_function",
      "params": {
        "file": "core/logc/logs.go",
        "source": "// Field returns a LogField for the given key and value.\nfunc Field(key string, value interface{}) LogField {\n\treturn logx.Field(key, value)\n}",
        "auto_organize_imports": true
      }
    },
    {
      "tool": "semantic_insert_function",
      "params": {
        "file": "core/logc/logs.go",
        "source": "// Info writes v into access log.\nfunc Info(ctx context.Context, v ...interface{}) {\n\tgetLogger(ctx).Info(v...)\n}",
        "auto_organize_imports": true
      }
    },
    {
      "tool": "semantic_insert_function",
      "params": {
        "file": "core/logc/logs.go",
        "source": "// Infof writes v with format into access log.\nfunc Infof(ctx context.Context, format string, v ...interface{}) {\n\tgetLogger(ctx).Infof(format, v...)\n}",
        "auto_organize_imports": true
      }
    },
    {
      "tool": "semantic_insert_function",
      "params": {
        "file": "core/logc/logs.go",
        "source": "// Infov writes v into access log with json content.\nfunc Infov(ctx context.Context, v interface{}) {\n\tgetLogger(ctx).Infov(v)\n}",
        "auto_organize_imports": true
      }
    },
    {
      "tool": "semantic_insert_function",
      "params": {
        "file": "core/logc/logs.go",
        "source": "// Infow writes msg along with fields into access log.\nfunc Infow(ctx context.Context, msg string, fields ...LogField) {\n\tgetLogger(ctx).Infow(msg, fields...)\n}",
        "auto_organize_imports": true
      }
    },
    {
      "tool": "semantic_insert_function",
      "params": {
        "file": "core/logc/logs.go",
        "source": "// Must checks if err is nil, otherwise logs the error and exits.\nfunc Must(err error) {\n\tlogx.Must(err)\n}",
        "auto_organize_imports": true
      }
    },
    {
      "tool": "semantic_insert_function",
      "params": {
        "file": "core/logc/logs.go",
        "source": "// MustSetup sets up logging with given config c. It exits on error.\nfunc MustSetup(c logx.LogConf) {\n\tlogx.MustSetup(c)\n}",
        "auto_organize_imports": true
      }
    },
    {
      "tool": "semantic_insert_function",
      "params": {
        "file": "core/logc/logs.go",
        "source": "// SetLevel sets the logging level. It can be used to suppress some logs.\nfunc SetLevel(level uint32) {\n\tlogx.SetLevel(level)\n}",
        "auto_organize_imports": true
      }
    },
    {
      "tool": "semantic_insert_function",
      "params": {
        "file": "core/logc/logs.go",
        "source": "// SetUp sets up the logx. If already set up, just return nil.\n// we allow SetUp to be called multiple times, because for example\n// we need to allow different service frameworks to initialize logx respectively.\n// the same logic for SetUp\nfunc SetUp(c LogConf) error {\n\treturn logx.SetUp(c)\n}",
        "auto_organize_imports": true
      }
    },
    {
      "tool": "semantic_insert_function",
      "params": {
        "file": "core/logc/logs.go",
        "source": "// Slow writes v into slow log.\nfunc Slow(ctx context.Context, v ...interface{}) {\n\tgetLogger(ctx).Slow(v...)\n}",
        "auto_organize_imports": true
      }
    },
    {
      "tool": "semantic_insert_function",
      "params": {
        "file": "core/logc/logs.go",
        "source": "// Slowf writes v with format into slow log.\nfunc Slowf(ctx context.Context, format string, v ...interface{}) {\n\tgetLogger(ctx).Slowf(format, v...)\n}",
        "auto_organize_imports": true
      }
    },
    {
      "tool": "semantic_insert_function",
      "params": {
        "file": "core/logc/logs.go",
        "source": "// Slowv writes v into slow log with json content.\nfunc Slowv(ctx context.Context, v interface{}) {\n\tgetLogger(ctx).Slowv(v)\n}",
        "auto_organize_imports": true
      }
    },
    {
      "tool": "semantic_insert_function",
      "params": {
        "file": "core/logc/logs.go",
        "source": "// Sloww writes msg along with fields into slow log.\nfunc Sloww(ctx context.Context, msg string, fields ...LogField) {\n\tgetLogger(ctx).Sloww(msg, fields...)\n}",
        "auto_organize_imports": true
      }
    },
    {
      "tool": "semantic_insert_function",
      "params": {
        "file": "core/logc/logs.go",
        "source": "// getLogger returns the logx.Logger with the given ctx and correct caller.\nfunc getLogger(ctx context.Context) logx.Logger {\n\treturn logx.WithContext(ctx).WithCallerSkip(1)\n}",
        "auto_organize_imports": true
      }
    },
    {
      "tool": "semantic_organize_imports",
      "params": {
        "file": "core/logc/logs.go",
        "add": [
          "github.com/zeromicro/go-zero/core/logx"
        ]
      }
    }
  ]
}
```
