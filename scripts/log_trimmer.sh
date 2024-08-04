#!/bin/sh

# log_trimmer.sh
# Trims a log file to the last n lines

LOG_FILE="/path/to/your/logfile.log"
MAX_LINES=100  # Keep the last 100 lines

# Check if the log file exists
if [ -f "$LOG_FILE" ]; then
    # Trim the log file
    tail -n $MAX_LINES "$LOG_FILE" > "$LOG_FILE.tmp"
    mv "$LOG_FILE.tmp" "$LOG_FILE"
    echo "Log file trimmed to the last $MAX_LINES lines."
else
    echo "Log file does not exist: $LOG_FILE"
fi

# Open the crontab with crontab -e and add the following line to run the script every hour:
# 0 * * * * /path/to/log_trimmer.sh

# Ensure the cron service is running by executing sudo /etc/init.d/crond restart.



