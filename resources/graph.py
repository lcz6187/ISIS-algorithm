import numpy as np
import matplotlib.pyplot as plt
import glob

# Consolidate all log files matching the pattern
files = glob.glob("*_processing.log")

# Group processing times by transaction ID (to get max)
transaction_times = {}

for file in files:
    try:
        # Load processing times from CSV 
        # Format: msg_id,processing_time_in_seconds
        data = np.genfromtxt(file, delimiter=',', dtype=None, encoding=None)

        # In case a file is empty or contains one entry, ensure it's an array.
        if data.ndim == 0:
            data = [data]

        for row in data:
            msg_id, proc_time = row
            if msg_id in transaction_times:
                transaction_times[msg_id] = max(transaction_times[msg_id], proc_time)
            else:
                transaction_times[msg_id] = proc_time
    except Exception as e:
        print(f"Error reading {file}: {e}")

# Convert to numpy array and sort
max_times  = np.array(list(transaction_times.values())) * 1000 # Seconds to MS
max_times .sort()

# Generate percentiles from 1 to 99
percentiles = np.linspace(1, 99, 99)

# Calculate processing times at these percentiles
cdf_times = np.percentile(max_times , percentiles)

# Plot the CDF
plt.figure(figsize=(8, 6))
plt.plot(cdf_times, percentiles, marker='.', linestyle='-')
plt.xlabel('Transaction Processing Time (ms)')
plt.ylabel('Percentile (%)')
plt.title('CDF of Transaction Processing Time (ms)')
plt.xlim(left=0)
plt.ylim(1, 99)
plt.grid(True)
# plt.show()

# Save
plt.savefig("graph.png", dpi=300, bbox_inches='tight')
