import argparse
import os
import re
from collections import Counter

def generate_html(ips, agents, statuses, output_file):
    with open(output_file, "w") as f:
        f.write("<html><head><title>Log Report</title></head><body>")
        f.write("<h1>Log analysis</h1>")

        f.write("<h2>Top IPs</h2><ul>")
        for ip, count in ips.most_common(args.top):
            f.write(f"<li>{ip} — {count}</li>")
        f.write("</ul>")

        f.write("<h2>Top User-Agent</h2><ul>")
        for ua, count in agents.most_common(args.top):
            f.write(f"<li>{ua} — {count}</li>")
        f.write("</ul>")

        f.write("<h2>HTTP statuses</h2><ul>")
        for status, count in statuses.items():
            f.write(f"<li>{status} — {count}</li>")
        f.write("</ul>")

        f.write("</body></html>")

parser = argparse.ArgumentParser(description="Log analyzer for Apache/Nginx")
parser.add_argument("-f", "--file", required=True, help="Path to access.log")
parser.add_argument("-o", "--output", default="report.html", help="HTML report file")
parser.add_argument("--top", type=int, default=10, help="Number of top IPs/UserAgents to show")
args = parser.parse_args()

if not os.path.exists(args.file):
    print(f"ERROR: No such file '{args.file}' ")
    exit(1)

 log_pattern = re.compile(
    r'(?P<ip>\d+\.\d+\.\d+\.\d+)\s'  # IP
    r'.*\['
    r'(?P<time>[^\]]+)\]\s'  # час
    r'"(?P<method>\w+)\s(?P<url>\S+)\s(?P<protocol>[^"]+)"\s'  # метод, URL, протокол
    r'(?P<status>\d{3})\s'  # статус код
    r'(?P<size>\d+|-)\s'  # розмір
    r'"(?P<referer>[^"]*)"\s'  # реферер
    r'"(?P<user_agent>[^"]*)"'  # User-Agent
)


ips = Counter()
statuses = Counter()
agents = Counter()

with open(args.file, 'r') as f:
    for line in f:
        match = log_pattern.match(line)
        if match:
            log = match.groupdict()
            ips[log["ip"]] += 1
            statuses[log["status"]] += 1
            agents[log["user_agent"]] += 1

generate_html(ips, agents, statuses, args.output)

for ip, count in ips.most_common(args.top):
    print(f"{ip}: {count}")
for status, count in statuses.items():
    print(f"{status}: {count}")
for ua, count in agents.most_common(args.top):
    print(f"{ua}: {count}")
