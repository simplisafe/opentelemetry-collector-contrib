# ID Collector Processor

This is a custom processor which is designed to help work around some limitations when writing OTLP logs to a Loki backend. When ingesting OTLP logs in Loki, all Attributes and Resource Attributes get converted to Loki structured metadata fields, which is great for readability, but since LogQL has no way to search across all structured metadata fields, it makes it very hard to find records with specific IDs that might be in any field.

This processor extracts IDs from all OTel Attributes, using a list of configurable regular expressions, joins them into a comma-delimited list, and adds them to a new OTel Attribute (the name of which is configurable).

Then, you can use this field to search in LogQL:

```
{service_name="myservice",env="prd"} | extracted_ids=~".*12345678.*"
```

## Configuration

```yaml
processors:
  idcollector:
    target_attribute: extracted_ids
    patterns:
      - \b[a-zA-Z0-9]{32}\b # 32 character alphanumeric IDs
      - \b[a-zA-Z0-9]{7,8}\b # 7 or 8 character alphanumeric IDs
    negative_patterns:
      - "notanid"
      - "alsonotanid"
```