SET search_path = hospital;

WITH Appointment_diagnosis AS (
    SELECT 
        patient_id, 
        diagnosis
    FROM 
        Appointments
    INNER JOIN 
        Conclusions 
    ON 
        Appointments.appointment_id = Conclusions.appointment_id
),
Patient_diagnosis AS (
    SELECT 
        EXTRACT(YEAR FROM AGE(birth_date)) AS age, 
        diagnosis
    FROM 
        Patients
    INNER JOIN 
        Appointment_diagnosis 
    ON 
        Appointment_diagnosis.patient_id = Patients.patient_id
)
SELECT 
    diagnosis
FROM 
    Patient_diagnosis
GROUP BY 
    diagnosis
HAVING 
    AVG(age) > 30
ORDER BY 
    diagnosis;