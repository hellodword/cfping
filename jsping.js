async function pingOnce(url, status){
    let now = +new Date();
    status = status || 200;
    let r = await fetch(url, {
        cache: 'no-store', 
        headers: {'Connection': 'close'},
    });
    if (r.status != status) {
        throw ('status '+ r.status);
    }
    return +new Date() - now;
}

async function ping(url, status, every) {
    every = every || 5;
    let all = [];
    for (let i = 0; i < every; i++) {
        all.push(await pingOnce(url, status));
    }
    console.log(all);
}