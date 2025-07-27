import nltk
from flask import Flask, request, jsonify
from flask_sqlalchemy import SQLAlchemy
from sqlalchemy import text
from nltk.stem import PorterStemmer

app = Flask(__name__)
app.config['SQLALCHEMY_DATABASE_URI'] = 'postgresql://adimn:admin@localhost/searchEngine'
app.config['SQLALCHEMY_TRACK_MODIFICATIONS'] = False
db = SQLAlchemy(app)

nltk.download('punkt_tab')

stemmer = PorterStemmer()

def call_search_with_levenshtein(query_terms, max_distance=2, penalty_factor=0.1, limit_results=10):
    sql = text("""
        SELECT * FROM search_with_levenshtein(:query_terms, :max_distance, :penalty_factor, :limit_results);
    """)

    result = db.session.execute(sql, {
        'query_terms': query_terms,
        'max_distance': max_distance,
        'penalty_factor': penalty_factor,
        'limit_results': limit_results
    })

    rows = result.fetchall()

    return [
        {'doc_url': row.doc_url, 'total_score': row.total_score}
        for row in rows
    ]

@app.route('/search')
def search():
    query = request.args.get('q', '')
    if not query:
        return jsonify({'error': 'Missing query'}), 400

    words = nltk.word_tokenize(query.lower())
    stemmed_words = [stemmer.stem(w) for w in words]

    results = call_search_with_levenshtein(stemmed_words)
    return jsonify(results)


if __name__ == '__main__':
    app.run(debug=True)