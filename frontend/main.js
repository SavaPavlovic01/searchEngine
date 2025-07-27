function keydown(event) {
    if (event.key === 'Enter') {
        const query = this.value.trim();
        if (!query) return;

        console.log('Searching for:', query);

        fetch(`http://localhost:5000/search?q=${encodeURIComponent(query)}&ts=true`)
            .then(response => {
                if (!response.ok) throw new Error("Network response was not ok");
                return response.json();
            })
            .then(data => {
                const output = document.getElementById('output');
                output.innerHTML = ''; 

                if (data.results?.length === 0) {
                    output.innerHTML = '<p>No results found.</p>';
                    return;
                }
                console.log(data)
                data.forEach(result => {
                    const div = document.createElement('div');
                    div.className = 'result';

                    div.innerHTML = `
                        <h3><a href="${result.doc_url}" target="_blank">${result.title || result.doc_url}</a></h3>
                        <p class="url">${result.doc_url}</p>
                        <p>${result.snippet || 'No snippet available.'}...</p>
                    `;

                    output.appendChild(div);
                });
            })
            .catch(error => {
                document.getElementById('output').innerHTML = `<p class="error">Error: ${error.message}</p>`;
            });
    }
}

document.addEventListener('DOMContentLoaded', () => {
  const input = document.getElementById('search');
  input.addEventListener('keydown', keydown);
  //window.addEventListener('keydown', keydown);
});
