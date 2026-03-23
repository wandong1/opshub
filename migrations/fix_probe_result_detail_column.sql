-- Fix probe_results.detail column to mediumtext to support large workflow result JSON
ALTER TABLE probe_results MODIFY COLUMN detail mediumtext;
