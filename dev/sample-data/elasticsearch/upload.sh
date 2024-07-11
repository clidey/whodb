curl -XPOST "http://localhost:9200/_bulk" -H "Content-Type: application/json" --data-binary @data.json


# where clause {"query":{"query_string":{"query":"*John*"}}}