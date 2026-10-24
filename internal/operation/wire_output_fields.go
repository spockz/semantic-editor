// Package operation centralizes repeated machine-output keys so public schemas and projectors stay aligned.
package operation

const (
	introducedField                     = "introduced"
	outputSnapshotIDField               = "snapshot_id"
	outputQualifiedNameField            = "qualified_name"
	editReceiptContinuationFilePath     = "result.edit_receipt.continuations[].file"
	editReceiptContinuationSymbolPath   = "result.edit_receipt.continuations[].symbol"
	editReceiptContinuationRevisionPath = "result.edit_receipt.continuations[].expected_revision"
)

const outputActiveCountField = "active_count"
