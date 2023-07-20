# Ancient Pruner

## Why
The chain freezer improves efficiency by moving the ancient chain items into their own flat files. This allows us to split the database into an active set, which requires a fast SSD disk, and an immutable freezer set, for which a slower HDD disk is sufficient.

However, for Geth running as a state node, ancient data is not needed and it's more beneficial to delete it directly. If the user enables "ancient.prune", the system will completely discard the ancient blocks instead of writing them to the freezer database.

