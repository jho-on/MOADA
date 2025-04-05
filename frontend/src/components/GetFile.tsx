import { useState } from "react"


function GetFile(){
    const [loading, setLoading] = useState<boolean>(false);
    const [idPublic, setIdPublic] = useState<string>("");

    const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        setIdPublic(e.target.value);
    }

    const handleSubmit = async () => {
        try{
            setLoading(true)

            if (!/^[a-fA-F0-9]{64}$/.test(idPublic)) {
                throw new Error("Invalid ID Format.");
            }

            const res = await fetch(`http://localhost:8082/downloadFile?idPublic=${encodeURIComponent(idPublic)}`);

            if (!res.ok){
                throw new Error("Failed to fetch the file.");
            }

            let filename = "downloaded-file";
            const contentDisposition = res.headers.get("Content-Disposition");

            if (contentDisposition){
                const regex = /filename="(.+)"/;
                const matches = regex.exec(contentDisposition);
                if (matches && matches[1]) {
                    filename = matches[1];
                }
            }


            const blob = await res.blob();
            const url = window.URL.createObjectURL(blob);

            const link = document.createElement("a");
            link.href = url;
            link.download = filename;
            link.click();
      
            window.URL.revokeObjectURL(url);

        }catch (error){
            console.error("Error:", error);
            alert("An error occurred while getting the file.");
        } finally{
            setLoading(false)
        }
    }   

    return(
        <>
            {loading && <h1>Downloading File</h1>}

            {!loading && (
                <form onSubmit={handleSubmit}>
                    <label htmlFor="">Public Id:</label>
                    <input type="text" name="" id="" onChange={handleInputChange}/>
                    <input className="button" type="submit" value="Get File"/>
                </form>
            )}
        </>
    )
}


export default GetFile