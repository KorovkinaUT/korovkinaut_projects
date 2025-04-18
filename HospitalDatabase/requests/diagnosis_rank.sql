SET search_path = hospital;

SELECT 
    diagnosis,
    AVG(EXTRACT(YEAR FROM AGE(birth_date))) AS avg_age,
    RANK() OVER (ORDER BY AVG(EXTRACT(YEAR FROM AGE(birth_date))) DESC) AS rank
FROM 
    Conclusions
INNER JOIN 
    Appointments 
    ON Conclusions.appointment_id = Appointments.appointment_id
INNER JOIN 
    Patients 
    ON Appointments.patient_id = Patients.patient_id
GROUP BY 
    diagnosis
ORDER BY 
    rank;