SET search_path = hospital;

SELECT 
    date_time::time AS time, 
    office
FROM 
    Appointments
WHERE 
    doctor_id = (
        SELECT 
            doctor_id
        FROM 
            Doctors
        WHERE 
            surname = 'Иванов' 
            AND name = 'Алексей'
    )
ORDER BY 
    time ASC;