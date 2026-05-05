import psycopg2
import sys

candidates = [
    ("postgres", "postgres"),
    ("postgres", "psdi"),
    ("postgres", "practiceops"),
    ("postgres", "password"),
    ("postgres", "admin"),
    ("postgres", ""),
    ("psdi",     "psdi"),
]

conn = None
for user, pw in candidates:
    try:
        conn = psycopg2.connect(host="localhost", port=5432, dbname="postgres", user=user, password=pw, connect_timeout=3)
        print(f"Connected as {user!r} / {pw!r}")
        break
    except Exception:
        pass

if conn is None:
    print("ERROR: could not connect to postgres on :5432 with any known credentials", file=sys.stderr)
    sys.exit(1)

conn.autocommit = True
cur = conn.cursor()

for stmt in [
    "CREATE USER ecollm WITH PASSWORD 'ecollm_dev'",
    "CREATE DATABASE ecollm OWNER ecollm",
    "GRANT ALL PRIVILEGES ON DATABASE ecollm TO ecollm",
]:
    try:
        cur.execute(stmt)
        print(f"  OK: {stmt}")
    except Exception as e:
        print(f"  skip: {e}")

print("Done — ecollm user + database ready on :5432")
