var regions = [];

async function init() {
  const response = await fetch('/api/regions')
  regions = await response.json()

  console.log("regions", regions)

  regions.forEach(region => monitor(region))
}

function monitor(region) {
  const tbody = document.querySelector('#tbl tbody')
  const tr = document.createElement('tr')
  const regionCell = document.createElement('td')
  regionCell.innerText = region.code
  tr.appendChild(regionCell)
  
  const valueCell = document.createElement('td')
  tr.appendChild(valueCell)
  
  const latencyCell = document.createElement('td')
  tr.appendChild(latencyCell)
  
  tbody.appendChild(tr)

  var sse = new EventSource('/api/stream?region=' + region.code);
  sse.addEventListener("update", function(e) {
    const data = JSON.parse(e.data)
    valueCell.innerText = data.value
    latencyCell.innerText = (data.latency * 1000).toFixed(3) + 'ms'
  })
}

function inc() {
  fetch('/api/inc', {method: 'POST'})
    .then(response => response.json())
    .then(data => {
      if (data.error !== undefined) throw data.error
    })
    .catch((error) => alert(error))
}

init()