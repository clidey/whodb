# Save MongoDB JSON Document Edits As Replacements

The MongoDB **Document JSON Editor** saves existing documents as replacement edits rather than `$set` patches.

This keeps the JSON editor aligned with the user model: the user is editing the document shape they see. Field additions stay where the user placed them, field removals remove stored fields, and top-level field order is preserved when a content change is submitted. Changing only top-level field order is treated as a no-op rather than a database edit. The original `_id` remains immutable and is kept as the identity filter for the replacement.

The **Collection Table View** keeps field-level patch behavior for inline scalar edits and **Field JSON Editor** edits, because those interactions target one field rather than the whole document.
