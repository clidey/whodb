//! Row hydration: stringly-typed wire results → native-typed rows.
//! Behavior is pinned by the shared cross-language conformance fixtures.

use serde_json::{Map, Value};

use crate::gen::hydration::{HYDRATION_DEFAULT, HYDRATION_RULES};

/// One hydrated record: column name → native-typed value.
pub type Row = Map<String, Value>;

fn rule_for(column_type: &str) -> &'static str {
    let lowered = column_type.to_lowercase();
    HYDRATION_RULES
        .iter()
        .find(|(name, _)| *name == lowered)
        .map(|(_, kind)| *kind)
        .unwrap_or(HYDRATION_DEFAULT)
}

/// Coerces one stringly-typed cell into its native type per the shared rules.
/// Non-string values pass through unchanged (the IPC transport delivers
/// natively-typed values). Timestamps stay RFC3339 strings on the Value level
/// but are tagged for the conformance canonical form by the caller.
pub(crate) fn coerce_value(raw: &Value, column_type: &str) -> Value {
    let text = match raw {
        Value::Null => return Value::Null,
        Value::String(text) => text,
        other => return other.clone(),
    };
    match rule_for(column_type) {
        "int" => text
            .parse::<i64>()
            .map(Value::from)
            .unwrap_or_else(|_| raw.clone()),
        "float" => text
            .parse::<f64>()
            .map(Value::from)
            .unwrap_or_else(|_| raw.clone()),
        "bool" => Value::Bool(text == "true" || text == "t" || text == "1"),
        "timestamp" | "date" => {
            // Rust has no std datetime type; hydrated timestamps carry the
            // canonical "@date:<RFC3339 Z>" marker used across the SDKs'
            // conformance protocol, letting callers detect and parse them
            // with their preferred time crate.
            if looks_like_rfc3339(text) || looks_like_date(text) {
                Value::String(format!("@date:{}", normalize_rfc3339(text)))
            } else {
                raw.clone()
            }
        }
        "json" => serde_json::from_str(text).unwrap_or_else(|_| raw.clone()),
        _ => raw.clone(),
    }
}

fn looks_like_rfc3339(text: &str) -> bool {
    text.len() >= 19
        && text.as_bytes().get(4) == Some(&b'-')
        && text.as_bytes().get(10) == Some(&b'T')
}

fn looks_like_date(text: &str) -> bool {
    text.len() == 10
        && text.as_bytes().get(4) == Some(&b'-')
        && text.as_bytes().get(7) == Some(&b'-')
}

fn normalize_rfc3339(text: &str) -> String {
    if looks_like_date(text) {
        return format!("{text}T00:00:00Z");
    }
    text.replace("+00:00", "Z")
}

struct WireColumn {
    name: String,
    column_type: String,
}

/// Normalizes the two wire result shapes:
/// DatasetQueryResult {columns: [name], rows, total} (no column types) and
/// CE-derived RowsResult {Columns: [{Name, Type}], Rows, TotalCount}.
fn normalize(result: &Value) -> (Vec<WireColumn>, Vec<Value>, Value) {
    if let Some(raw_columns) = result.get("columns").and_then(Value::as_array) {
        let columns = raw_columns
            .iter()
            .map(|column| WireColumn {
                name: column.as_str().unwrap_or_default().to_string(),
                column_type: String::new(),
            })
            .collect();
        let rows = result
            .get("rows")
            .and_then(Value::as_array)
            .cloned()
            .unwrap_or_default();
        return (
            columns,
            rows,
            result.get("total").cloned().unwrap_or(Value::Null),
        );
    }
    let columns = result
        .get("Columns")
        .and_then(Value::as_array)
        .map(|raw| {
            raw.iter()
                .map(|column| WireColumn {
                    name: column
                        .get("Name")
                        .and_then(Value::as_str)
                        .unwrap_or_default()
                        .to_string(),
                    column_type: column
                        .get("Type")
                        .and_then(Value::as_str)
                        .unwrap_or_default()
                        .to_string(),
                })
                .collect()
        })
        .unwrap_or_default();
    let rows = result
        .get("Rows")
        .and_then(Value::as_array)
        .cloned()
        .unwrap_or_default();
    (
        columns,
        rows,
        result.get("TotalCount").cloned().unwrap_or(Value::Null),
    )
}

/// Builds a property-type map (apiName → dataType) from entity metadata.
pub(crate) fn property_types_of(entity: &Value) -> Map<String, Value> {
    let mut types = Map::new();
    if let Some(properties) = entity.get("properties").and_then(Value::as_array) {
        for property in properties {
            if let (Some(api_name), Some(data_type)) = (
                property.get("apiName").and_then(Value::as_str),
                property.get("dataType").and_then(Value::as_str),
            ) {
                types.insert(api_name.to_string(), Value::String(data_type.to_string()));
            }
        }
    }
    types
}

/// Hydrates a wire result into native-typed rows. Ontology property metadata,
/// when supplied, overrides the wire column type.
pub(crate) fn hydrate_rows(
    result: &Value,
    property_types: &Map<String, Value>,
) -> (Vec<Row>, Value) {
    let (columns, raw_rows, total) = normalize(result);
    let rows = raw_rows
        .iter()
        .map(|raw_row| {
            let cells = raw_row.as_array().cloned().unwrap_or_default();
            let mut row = Row::new();
            for (index, column) in columns.iter().enumerate() {
                let column_type = property_types
                    .get(&column.name)
                    .and_then(Value::as_str)
                    .unwrap_or(&column.column_type);
                let cell = cells.get(index).cloned().unwrap_or(Value::Null);
                row.insert(column.name.clone(), coerce_value(&cell, column_type));
            }
            row
        })
        .collect();
    (rows, total)
}
