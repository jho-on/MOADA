import { useEffect, useState } from "react"

type UserData = {
    Ip: string;
    Files: Array<string>;
    FilesNumber: number;
    UsedSpace: number;
    IpSavedDate: string;
    IpExpireDate: string;
    APICalls: number;
    APILastCallDate: string;
};

function UserData(){
    const [loading, setLoading] = useState<boolean>(false);
    const [userData, setUserData] = useState<UserData | null>(null);
    
    
    useEffect(() => {
        const getData = async () => {
            try{
                setLoading(true)
                const res = await fetch("http://localhost:8082/myInfo");
                
                if (!res.ok){
                    alert("Error uploading file.");
                    return
                }

                const data = await res.json();
                setUserData(data.data);

            }catch (error){
                console.error("Error:", error);
                alert("An error occurred while getting user data.");
            }
            finally{
                setLoading(false);
            }
        }

        getData();

    }, [])

    return (
        <div>
            {loading && <h1>Loading Your Data</h1>}

            {userData && !loading && (
                <div>
                    <p>User IP: {userData.Ip}</p>
                    <p>Files Number: {userData.FilesNumber}</p>
                    <p>Used Space: {userData.UsedSpace}</p>
                    <p>API Calls: {userData.APICalls}</p>
                    <p>Last API Call Date: {userData.APILastCallDate}</p>
                    <p>IP Saved Date: {userData.IpSavedDate}</p>
                    <p>IP Expire Date: {userData.IpExpireDate}</p>
                    <h3>Files:</h3>
                    <ul>
                        {userData.Files.map((file, index) => (
                        <li key={index}>{file}</li>
                        ))}
                    </ul>
                </div>
            )}

        </div>
    )
}


export default UserData