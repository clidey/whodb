# Bound MongoDB Field Inference To A Document Sample

MongoDB collection table columns are inferred from a bounded document sample so collection browsing stays responsive on large collections. The resulting **Sampled Field Set** is intentionally not a complete schema; fields outside the sample can still appear through the current page or pending document changes.
