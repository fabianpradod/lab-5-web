// "Actualizar sin reload"
// solo actualizamos el numero del episodio en la tabla despues del fetch

async function nextEpisode(id) {
    // Mandamos POST al servidor con el id de la serie
    const response = await fetch(`/update?id=${id}`, { method: "POST" });

    if (response.ok) {
        location.reload();
    }
}

async function prevEpisode(id) {
    // Boton -1, manda POST a /update-prev?id=X
    const response = await fetch(`/update-prev?id=${id}`, { method: "POST" });

    if (response.ok) {
        location.reload();
    }
}

async function deleteSeries(id) {
    // Eliminar serie — solo cuenta si usas metodo DELETE
    const response = await fetch(`/delete?id=${id}`, { method: "DELETE" });

    if (response.ok) {
        location.reload();
    }
}