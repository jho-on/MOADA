# MongoDB Database Documentation
## Collection: fileMetadata

The fileMetadata collection stores information about files uploaded to the server. Each document represents a file with the following fields:

Document Fields:

    idPrivate (string):
    A unique identifier used to request the download of the file.

    idPublic (string):
    A unique identifier used to request the deletion of the file.

    name (string):
    The name of the file, including the extension (e.g., "document.pdf").

    size (double):
    The size of the file in bytes.

    savedDate (date):
    The date and time when the file was saved on the server.

    expireDate (date):
    The date and time when the file will be automatically deleted from the server. This is typically one day after the savedDate.

    email (string, optional):
    The email address to which a message will be sent when the file expires.

Example Document:

    {
        "idPrivate": "unique-private-id",
        "idPublic": "unique-public-id",
        "name": "document.pdf",
        "size": 204800,
        "savedDate": ISODate("2025-03-15T08:00:00Z"),
        "expireDate": ISODate("2025-03-16T08:00:00Z"),
        "email": "user@example.com"
    }

## Collection: users

The users collection stores information about users who have uploaded files to the server. Each document represents a user identified by their IP address. Below are the fields described in the schema.

Document Fields:

    ip (string):
    Anonymized IP address that made the upload request.

    files (array):
    A list of files uploaded by this IP. Each item in the list is a fileMetadata object.

    filesNumber (int):
    The total number of files currently on the server for this IP.

    usedSpace (double):
    The total space used (in bytes) by all files uploaded by this IP.

    ipSavedDate (date):
    The date when the user's IP was saved in the database.

    ipExpireDate (date):
    The date when the IP will be automatically removed from the database. This is usually one day after the last file upload.

    APICalls (int):
    The number of API calls made by this IP.

    APILastCallDate (date):
    The date and time of the last API call made by this IP.

Example Document:

    {
        "ip": "192.168.1.1",
        "files": [
            {
                "idPrivate": "unique-private-id",
                "idPublic": "unique-public-id",
                "name": "document.pdf",
                "size": 204800,
                "savedDate": ISODate("2025-03-15T08:00:00Z"),
                "expireDate": ISODate("2025-03-16T08:00:00Z"),
                "email": "user@example.com"
            }
        ],
        "filesNumber": 1,
        "usedSpace": 204800,
        "ipSavedDate": ISODate("2025-03-15T08:00:00Z"),
        "ipExpireDate": ISODate("2025-03-16T08:00:00Z"),
        "APICalls": 5,
        "APILastCallDate": ISODate("2025-03-15T09:00:00Z")
    }