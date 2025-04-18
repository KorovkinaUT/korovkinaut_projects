SET search_path = hospital;

SELECT 
    surname || ' ' || name AS full_name, 
    EXTRACT(YEAR FROM AGE(birth_date)) AS age
FROM 
    Patients
WHERE 
    patient_id IN (
        SELECT 
            patient_id
        FROM 
            Conclusions
        INNER JOIN 
            Appointments 
        ON 
            Appointments.appointment_id = Conclusions.appointment_id
        WHERE 
            diagnosis = 'Гастрит'
    )
ORDER BY 
    full_name ASC;