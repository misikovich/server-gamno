#!/bin/bash
# Usage: ./show_videos.sh your_db.db
sqlite3 "videos.db" \
  -cmd ".headers on" \
  -cmd ".mode column" \
  "SELECT * FROM videos;"
