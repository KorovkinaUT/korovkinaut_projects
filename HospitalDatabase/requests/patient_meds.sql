SET search_path = hospital;

WITH Patient AS (
    SELECT 
        patient_id
    FROM 
        Patients
    WHERE 
        surname = 'Соколов' 
        AND name = 'Иван'
)
SELECT DISTINCT 
    name AS medicine
FROM 
    Medicines
WHERE 
    medicine_id IN (
        SELECT 
            medicine_id
        FROM 
            Prescriptions
        INNER JOIN 
            Conclusions 
        ON 
            Conclusions.conclusion_id = Prescriptions.conclusion_id
        WHERE 
            appointment_id IN (
                SELECT 
                    appointment_id
                FROM 
                    Appointments
                WHERE 
                    patient_id = (SELECT patient_id FROM Patient)
            )
    )
ORDER BY 
    medicine ASC;