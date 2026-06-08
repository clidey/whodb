# Build MongoDB Table Columns From Visible Fields

MongoDB **Collection Table View** columns are built from the **Visible Field Set** so collection browsing reflects the documents currently on screen and any pending document changes. The view does not infer extra fields from a collection sample. The `_id` column remains first. Other visible fields are added in first-seen **Document Field Order** while scanning current visible documents from top to bottom; fields introduced only by pending document changes are appended afterward in change order.
