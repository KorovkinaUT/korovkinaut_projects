SET search_path = hospital;

SELECT 
    surname || ' ' || name AS full_name
FROM 
    Patients
WHERE 
    patient_id IN (
        SELECT 
            patient_id
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
    )
ORDER BY 
    full_name;