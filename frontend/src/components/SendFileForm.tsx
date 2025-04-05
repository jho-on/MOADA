import { useState } from "react";
import { formatDate } from "../utils/dateUtils";

type FileData = {
    idPublic: string;
    idPrivate: string;
    name: string;
    size: number;
    savedDate: string;
    expireDate: string;
    email: string;
};
  

function SendFileForm(){

    const [file, setFile] = useState<File | null>(null);
    const [loading, setLoading] = useState<boolean>(false);
    const [data, setData] = useState<FileData | null>(null); 

    const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0];

        if (file) {
            setFile(file);
        }
        console.log(file);
    };

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();

        if (!file) {
            alert("Please, enter a file before submiting");
        }else{
            const formData = new FormData();
            formData.append("file", file);

            try {
                setLoading(true);
                const response = await fetch("http://localhost:8082/sendFile", {
                    method: "POST",
                    body: formData,
                });
                
                if (response.ok) {
                    alert("File uploaded successfully!");

                    const data_ = await response.json();
                    console.log(data_.message);
                    setData(data_.data);
                } else {
                    alert("Error uploading file.");
                }

            } catch (error) {
                console.error("Error:", error);
                alert("An error occurred while uploading the file.");

            } finally{
                setLoading(false)
            }
        }
    };

    return(

        <div>
            {loading && <h1>Loading</h1>}
            {

            data ? (
                <div>
                    <h2>File Uploaded Successfully</h2>
                    <p>Public Id: {data.idPublic}</p>
                    <p>Private Id: {data.idPrivate}</p>
                    <p>Expire Date: {formatDate(data.expireDate)}</p>
                </div>
            ) : (
                !loading && (
                    <>
                        <form onSubmit={handleSubmit}>
                            <label>Select a file:</label>
                            <input  type="file" onChange={handleFileChange}/>
                            <input className="button" type="submit" value="Send File"/>
                        </form>
                        <p>When sending files you agree with our <a href=""><b>Terms of Service</b></a></p>
                    </>
                )
            )
            }
        </div>

       
    )
}


export default SendFileForm